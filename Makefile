build:
	go build -o bin/app .

run:
	go run *.go -src=/foo.jpg -dst=./out.jpg -debug
