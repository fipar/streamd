all: 
	go build streamd.go

run: all
	tail -f /var/log/system.log |./streamd  -port=9898 -debug=true

clean: 
	rm -f streamd 
