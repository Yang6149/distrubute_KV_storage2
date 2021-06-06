debug
go run .\main.go -t -rP 8000 -P 8100 -m 0 -ismaster
go run .\main.go -t -rP 8001 -P 8101 -m 1 -ismaster
go run .\main.go -t -rP 8002 -P 8102 -m 2 -ismaster

go run .\main.go -t -rP 8200 -P 8210 -m 0 -g 1
go run .\main.go -t -rP 8201 -P 8211 -m 1 -g 1
go run .\main.go -t -rP 8202 -P 8212 -m 2 -g 1

go run .\main.go -t -rP 8300 -P 8310 -m 0 -g 2
go run .\main.go -t -rP 8301 -P 8311 -m 1 -g 2
go run .\main.go -t -rP 8302 -P 8312 -m 2 -g 2

go run .\main.go -t -rP 8400 -P 8410 -m 0 -g 3
go run .\main.go -t -rP 8401 -P 8411 -m 1 -g 3
go run .\main.go -t -rP 8402 -P 8412 -m 2 -g 3

No connection could be made because the target machine actively refused it.