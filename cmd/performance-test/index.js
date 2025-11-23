import grpc from 'k6/net/grpc';
import {check, sleep} from 'k6';

const client = new grpc.Client();

export const options = {
    scenarios: {
        constant_request_rate: {
            executor: 'constant-arrival-rate',
            rate: 5000,       // 1000 iters per second
            timeUnit: '1s',
            duration: '15s',
            preAllocatedVUs: 10,
        },
    },
};

// console.log('Loading proto...');
client.load(null, '../../pkg/bloomproto/bloom.proto');

const targets = [
    'localhost:8000',
    'localhost:8001',
    'localhost:8002',
    'localhost:8004',
]

function* roundRobin(items) {
    if (!items.length) return;

    let i = 0;
    while (true) {
        yield items[i];
        i = (i + 1) % items.length;
    }
}

const rrGen = roundRobin(targets);


export default function () {
    try {
        // 1. CONNECT
        client.connect(rrGen.next().value, {
            plaintext: true, // 👈 IMPORTANT: assume server is non-TLS
        });

        // 2. PREPARE REQUEST
        const data = {
            key: '7b84e6b5-82ea-47a6-bc6a-bc019fb69fad',
        };

        // 3. INVOKE RPC
        const res = client.invoke(
            'bloom_proto.DistributedBloomFilter/Test', // 👈 service + method
            data,
        );

        // 4. CHECK & LOG
        check(res, {
            'gRPC call succeeded': (r) => r && r.status === grpc.StatusOK,
        });

        if (!res || res.status !== grpc.StatusOK) {
            console.log('gRPC error:', JSON.stringify(res, null, 2));
        } else {
            // comment out in real load, keep for debugging
            // console.log('Response:', JSON.stringify(res.message));
        }
    } catch (e) {
        errorCount++
        // console.log('Exception in VU iteration:', String(e));
    } finally {
        // 5. CLOSE CONNECTION
        client.close();
    }

    sleep(0.001); // tiny sleep so logs are readable
}
