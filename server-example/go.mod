module mrpg/main

go 1.24.3

replace gserver => ../

require gserver v0.0.0-00010101000000-000000000000

require github.com/mattn/go-sqlite3 v1.14.31

require github.com/gorilla/websocket v1.5.3 // indirect
