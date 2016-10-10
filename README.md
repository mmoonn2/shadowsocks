# shadowsocks

## Run

* Server

```
./shadowsocks-server --config sample-server.json
```

* Client 

You can use the client binary file as follow:

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

then change the socket v5 proxy on your conputer network setting to 127.0.0.1:1081

**OR use shadowsocks-gui client**

shadowsocks-gui for mac
[ShadowsocksX-2.6.3.dmg](https://github.com/shadowsocks/shadowsocks-iOS/releases/download/2.6.3/ShadowsocksX-2.6.3.dmg)
[ShadowsocksX-NG-1.2.dmg](https://github.com/shadowsocks/ShadowsocksX-NG/releases/download/1.2/ShadowsocksX-NG-1.2.dmg)