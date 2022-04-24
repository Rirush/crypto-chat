# Crypto Chat

A very simple client-server application for sending signed messages between browsers using Web Crypto and WebSockets.

## Why?

I wanted to poke at Web Crypto and do something with it.

## Is it secure?

It looks good, but I'm not a cryptography expert, so probably not.

## How to run it

Make sure you have Go 1.18+ installed.

To build and start the server:
```bash
go build -o server ./cmd/server
./server
```

No additional setup necessary, but there are some possible issues you might encounter:

* Because this uses Web Crypto, in order to make this page functional, you need to have TLS working on non-localhost origins. 
This server doesn't support TLS, so you'll need to have a reverse-proxy that provides TLS. (that'll also require updating the frontend to use `wss://` instead of `ws://` protocol for WebSocket communication)

* There's no way to export/import keys. If you refresh the page, the keys and message history are lost.

* There's no way change listen address or port without modifying code, so you either have to accept that it runs on `0.0.0.0:8080` or change the code.

* It's inconvenient and extremely basic.

## Screenshot

<img width="1381" alt="image" src="https://user-images.githubusercontent.com/14316104/164974713-3bb96b95-fe03-439e-bc34-fc0be5814772.png">
