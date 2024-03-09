import http.server
import socketserver

PORT = 8000
JSON_FILE = "flagids.json"

Handler = http.server.SimpleHTTPRequestHandler

with socketserver.TCPServer(("", PORT), Handler) as httpd:
    print(f"Serving JSON file at http://localhost:{PORT}/{JSON_FILE}")
    httpd.serve_forever()
