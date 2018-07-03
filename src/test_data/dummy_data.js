/* 
 * This file is part of Flower.
 * 
 * Copyright ©2018 Nicolò Mazzucato
 * Copyright ©2018 Antonio Groza
 * Copyright ©2018 Brunello Simone
 * Copyright ©2018 Alessio Marotta
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
 * 
 * Flower is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 * 
 * Flower is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * 
 * You should have received a copy of the GNU General Public License
 * along with Flower.  If not, see <https://www.gnu.org/licenses/>.
 */

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
