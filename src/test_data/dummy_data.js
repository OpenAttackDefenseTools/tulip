export default {
    dummy_flows: [
        {
            id: "1",
            src: "127.0.0.1:45384",
            dst: "127.0.0.1:4003",
            data: [
                {
                    data:
                        "GET /mmm HTTP/1.1\r\nHost: localhost:4003\r\nConnection: keep-alive\r\nAccept-Encoding: gzip, deflate\r\nAccept: */*\r\nUser-Agent: python-requests/2.18.4\r\n\r\n",
                    actor: "server"
                },
                {
                    data:
                        "HTTP/1.0 500 Internal Server Error\r\nServer: BaseHTTP/0.6 Python/3.6.5\r\nDate: Wed, 13 Jun 2018 22:54:28 GMT\r\nContent-type: text/html\r\n\r\n",
                    actor: "client"
                },
                {
                    data: "nope",
                    actor: "client"
                }
            ]
        },
        {
            id: "2",
            src: "127.0.0.1:47098",
            dst: "127.0.0.1:4005",
            data: [
                {
                    data:
                        "GET /mmm HTTP/1.1\r\nHost: localhost:4005\r\nConnection: keep-alive\r\nAccept-Encoding: gzip, deflate\r\nAccept: */*\r\nUser-Agent: python-requests/2.18.4\r\n\r\n",
                    actor: "server"
                },
                {
                    data:
                        "HTTP/1.0 500 Internal Server Error\r\nServer: BaseHTTP/0.6 Python/3.6.5\r\nDate: Wed, 13 Jun 2018 22:54:28 GMT\r\nContent-type: text/html\r\n\r\n",
                    actor: "client"
                },
                {
                    data: "nope",
                    actor: "client"
                }
            ]
        },
        {
            id: "3",
            src: "127.0.0.1:50226",
            dst: "127.0.0.1:4001",
            data: [
                {
                    data:
                        "GET /mmm HTTP/1.1\r\nHost: localhost:4002\r\nConnection: keep-alive\r\nAccept-Encoding: gzip, deflate\r\nAccept: */*\r\nUser-Agent: python-requests/2.18.4\r\n\r\n",
                    actor: "server"
                }
            ]
        }
    ]
};
