# fileMonitor
This tool monitors changes to monitored paths.  
Notifies when the specified file has been created, removed, renamed, rewritten, or changed in permission.


## Installation
```
$ go get github.com/XapiMa/fileMonitor/cmd/fileMonitor
```

or

```
$ git clone https://github.com/XapiMa/fileMonitor.git
$ go build ./cmd/fileMonitor
```


## Usage
### Write config.yml
Write config.yml that defines the path to be monitored, the depth to search recursively, and the event to be monitored.  

- path
    - can not be omitted
- depth
    - default 0
    - When depth is -1, it recursively infinitely
- event
    - default create|remove|write|rename|permission
    - The available events are:
        - create
        - remove
        - rename
        - write
        - permission
    - You can set multiple events separated by `|`


ex.
```
- path: path/to/file
  depth: 0
  event: remove|write|rename|permission
- path: path/to/dire1/
  depth: 1
  event: remove|rename|permission
- path: path/to/dire2/
  depth: -1
```

## Execution

```
$ fileMonitor -t path/to/config.yml
```

If you want to write the result to a file:
```
$ webStatusChecker -t path/to/config.yml -o path/to/output/file
```

```
Usage of fileMonitor:
    -o string
            output file path. If not set, it will be output to standard output
    -t string
            path to config.yml (default "In the same directory as the executable file")
```


## Problem
In darwin the following command can not be judged correctly
- `mkdir -p a/b/c`
- `mkdir a a/b a/b/c`
- `rm -rf a` n the directory structure `a/b/c`
    - This case judged as follows
        - remove a/b/c
        - modify a/b
        - modify a/b/c
