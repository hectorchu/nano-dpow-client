nano-dpow-client
================

This is a websocket-based client for the NANO Distributed Proof-of-Work (DPoW) system.

This server understands the `work_generate` RPC call.

Install
-------

    go get -u github.com/hectorchu/nano-dpow-client

Usage
-----

    -f string
          Fallback RPC URL
    -k string
          API key
    -p int
          Listen port (default 7076)
    -u string
          User
