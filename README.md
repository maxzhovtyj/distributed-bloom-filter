ssh -i .ssh/digitalocean root@161.35.74.38
ssh -i .ssh/digitalocean root@134.209.244.237
ssh -i .ssh/digitalocean root@46.101.123.0

```shell
pm2 stop bloom-node;pm2 start /root/distributed-bloom-filter/bin/node-linux-amd64 --name="bloom-node"
```

```shell
pm2 stop zookeeper;pm2 start /root/distributed-bloom-filter/bin/zookeeper-linux-amd64 --name="zookeeper" -- -config=/root/distributed-bloom-filter/cmd/zookeeper/config.yml
```


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

### Node Exporter installation
```shell
sudo wget https://github.com/prometheus/node_exporter/releases/download/v1.10.2/node_exporter-1.10.2.linux-amd64.tar.gz
sudo tar xzf node_exporter-1.10.2.linux-amd64.tar.gz
root@ubuntu-s-1vcpu-2gb-fra1-03:~# sudo rm -rf node_exporter-1.10.2.linux-amd64.tar.gz
root@ubuntu-s-1vcpu-2gb-fra1-03:~# sudo mv node_exporter-1.10.2.linux-amd64 /etc/node_exporter
root@ubuntu-s-1vcpu-2gb-fra1-03:~# vim /etc/systemd/system/node_exporter.service
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
ExecStart=/etc/node_exporter/node_exporter
Restart=always

[Install]
WantedBy=multi-user.target
root@ubuntu-s-1vcpu-2gb-fra1-03:~# sudo systemctl daemon-reload
root@ubuntu-s-1vcpu-2gb-fra1-03:~# sudo systemctl enable node_exporter
Created symlink /etc/systemd/system/multi-user.target.wants/node_exporter.service → /etc/systemd/system/node_exporter.service.
root@ubuntu-s-1vcpu-2gb-fra1-03:~# sudo systemctl restart node_exporter
root@ubuntu-s-1vcpu-2gb-fra1-03:~#
```