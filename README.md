# shadowsocks

## Run

* Server

```
./shadowsocks-server --config sample-server.json
```

* Client

edit `sample-client.json` change `127.0.0.1` to specific server IP:

```
{
    "log_file": "./log/shadowsocks-client.log",
    "log_level": "debug",
    "log_max_days": 3,
	"local_port": 1081,
	"server_password": [
		["127.0.0.1:8387", "foobar"],
		["127.0.0.1:8388", "barfoo", "aes-128-cfb"]
	]
}
```

```
./shadowsocks-client --config sample-client.json
```