bin:
	mkdir -p bin

bin/cpu: bin
	go build -o bin/cpu ./cpu

bin/kernel: bin
	go build -o bin/kernel ./kernel

bin/memoria: bin
	go build -o bin/memoria ./memoria

bin/io: bin
	go build -o bin/io ./io

cpu: bin/cpu
kernel: bin/kernel
memoria: bin/memoria
io: bin/io

build: cpu kernel memoria io

clean:
	rm -f bin/*