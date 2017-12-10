.PHONY: test
test:
	go test -v -i
	go test -v

.PHONY: fuzz
fuzz:
	mkdir -p bin
	go-fuzz-build -o=bin/css-fuzz.zip github.com/ericchiang/css
	mkdir -p workdir/corpus
	./scripts/init-fuzz-corpus
	go-fuzz -bin=bin/css-fuzz.zip -workdir=workdir 

.PHONY: clean
clean:
	rm -rf bin
	rm -rf workdir
