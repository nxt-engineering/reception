all: reception

.PHONY: clean run

reception:
	$(MAKE) -C http
	go build reception.go

clean:
	$(MAKE) -C http clean
	rm -f reception

run:
	go run reception.go