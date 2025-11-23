

Single bloom
```shell
echo "GET http://localhost:8000/test?uid=32202899-2c89-419c-9837-d3f029708695" | vegeta attack -rate=4000/s -duration=15s -output=results_node.bin && cat results_node.bin | vegeta report
```

Requests      [total, rate, throughput]         60000, 4000.13, 4000.11
Duration      [total, attack, wait]             15s, 14.999s, 92.541µs
Latencies     [min, mean, 50, 90, 95, 99, max]  30.583µs, 156.711µs, 72.467µs, 164.301µs, 334.896µs, 1.643ms, 22.109ms
Bytes In      [total, mean]                     3300000, 55.00
Bytes Out     [total, mean]                     0, 0.00
Success       [ratio]                           100.00%
Status Codes  [code:count]                      200:60000  
Error Set:

```shell
cat results_node.bin | vegeta plot --title="Standard Bloom Filter" > node.html
```

Distributed
```shell
echo "GET http://localhost:8000/distributed/test?uid=32202899-2c89-419c-9837-d3f029708695" | vegeta attack -rate=4000/s -duration=15s -output=results_dbf.bin && cat results_dbf.bin | vegeta report
```