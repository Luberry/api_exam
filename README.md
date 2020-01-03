# SCOIR Technical Interview for Back-End Engineers
This repo contains an exercise intended for Back-End Engineers.

## Instructions
1. Fork this repo.
1. Using technology of your choice, complete [the assignment](./Assignment.md).
1. Update this README with
    * a `How-To` section containing any instructions needed to execute your program.
    * an `Assumptions` section containing documentation on any assumptions made while interpreting the requirements.
1. Before the deadline, submit a pull request with your solution.

## Expectations
1. Please take no more than 8 hours to work on this exercise. Complete as much as possible and then submit your solution.
1. This exercise is meant to showcase how you work. With consideration to the time limit, do your best to treat it like a production system.

## How-TO
There are two methods to run this project, the preferred environment is using `Docker`
### Docker
On a machine with the latest Docker installed (see the docs (here)[https://docs.docker.com/install/] for installation instructions)
you must run the following commands to build and run the project with docker
1. clone the source repository
```bash
git clone https://github.com/luberry/api_exam.git
```
2. cd into the repository and build the dockerfile
```bash
cd /path/to/api_exam
docker build -t api_exam .
```
3. now we can run the docker container, and we can mount our input, output and errors folders as volumes
```bash
docker run -v $PWD/input:/input \
           -v $PWD/errors:/errors \
           -v $PWD/output:/output api_exam
```

### Native Go Binary
You can also build this natively on your computer using Go. (See the upstream documentation (here)[https://golang.org/doc/install] for installation instructions)

1. clone the source repository
```bash
git clone https://github.com/luberry/api_exam.git
```

2. cd into the repository, download dependencies and build the binary
```bash
cd /path/to/api_exam
# download dependencies
go mod download
# build project
go build
```

3. run the built binary
    * Usage of ./api_exam:
        - error-directory="./errors": directory to output error files to
        - input-directory="./input": directory to watch for new `csv` files.
        - log-level="info": log level can be one of (panic,error,warn,info,debug,trace)
        - output-directory="./output": directory to output json files to
```bash
./api_exam -error-directory=/path/to/errors \
           -input-directory=/path/to/input \
           -output-directory=/path/to/output \
           -log-level=info
```

## Assumptions
While interpreting the requirements the following assumptions were made.
* The 8 hour time limit does not need to be in one sitting, and I was too anxious to start, 
but also had prior engagements in between so it was completed in two sessions spanning about 3-4 hours including documentation.
* The assumption was made that the `input`, `error`, and `output` directories exist prior to running the program.
* The errors file should only contain Row errors, not file open errors and such
