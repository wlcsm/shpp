# Simple Shell Preprocessor

Executes all text inside '%{' '}%' delimiters in the source text with a given program (default: `/bin/sh`) and inserts the stdout and stderr back into the original text.

More specifically, the text blocks will be given to the program as a positional argument and executed as

```
<prog> -c <text> [<args> ... ]
```

Inspired by the rc templating language https://werc.cat-v.org/docs/rc-template-lang.

## Installation

```
go install github.com/wlcsm/shpp@latest
```

## Configuration

* `SHPP_COMMAND`  The command used to execute the codeblock (default: `/bin/sh`)

## Examples

Basic usage will be to include the templated file as a positional argument

```
$ cat template.html
<p>%{echo "Hello, world}%</p>

$ shpp template.html
<p>Hello, world</p>
```

shpp will read from stdin if no arguments are supplied.

```
$ shpp < template.html
<p>Hello, world</p>
```

When the code inside the delimiters is executed, it will have access to STDIN, environment variables, and positional arguments of the parent process.

Passing via stdin

```
$ cat template.html
<p>%{ cat }%</p>

$ echo 'Hello World' | shpp template.yaml
<p>Hello, world</p>
```

Passing via environment variables

```
$ echo '%{echo $MSG}%' | MSG='Hello, world' ./shpp
Hello, world
```

Passing via positional arguments

```
$ cat template.html
<p>%{ echo $1, $2 }%</p>

$ echo 'Hello World' | shpp template.yaml 'Hello' 'world'
<p>Hello, world</p>
```

The "-" character may be used in place of the source file name to signify the template should be read from STDIN. This is useful when needed to provide positional arguments.

```
$ echo '%{printf $1 $2}%' | shpp - 'Hello,' 'world'
Hello, world
```
