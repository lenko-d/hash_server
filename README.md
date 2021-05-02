# Hash Server

## Hash Server provides hashing and encoding services

## Quick start

Follow the steps below in order to install dependencies, build the application and run test cases.

NOTE: The following examples use commands that are specific to Arch-based Linux distributions. For other Linux distributions or other operating systems use the corresponding equivalents.

### Install dependencies

* Install the Go programming language

```
pacman -S go
```

* Install the Make build automation tool

```
pacman -S make
```

* Install Apache Bench for testing purposes

```
pacman -S apache-tools
```


### Build the application:

```
make
```
The command above will generate the following binary in the current directory:

```
hash_server
```



### Run automated tests using Apache Bench(ab)

```
Start the server
./hash_server

In a new terminal window run the tests:
make test

```

### Example usage

```
Submit data to be hashed:
curl --data "password=testPassword"   http://localhost:8080/hash
the above returns <hash-id> that can be used to retrieve the hash.

Get the hashed data:
curl http://localhost:8080/hash/<hash-id>

Generate stats:
curl http://localhost:8080/stats
```


