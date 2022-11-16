# Simple Shell Preprocessor

Runs `sh` on all the code inside blocks delimited by `%{` `}%` in the source. Inspired by the rc templating language https://werc.cat-v.org/docs/rc-template-lang.

WARNING: Still in testing, do not use for production systems.

The entire contents of the embedded script is run by passing it as the first argument to `sh -c`. This means that it will potentially show up in the process list.

## Installation

```
go build
mv ./shpp /usr/local/bin
```

## Examples

Suppose we have the unprocessed file

```html
# index.html

<title>%{printf $NAME}%</title
<ul>
%{
for i in a b c; do
   echo '<li>'$i'</li>'
done
}%
</ul>
```

We can provide the variable `NAME` as an environment variable, and pass the file via STDIN

```
NAME=myname shpp < index.html
```

which produces

```html
<title>myname</title>
<ul>
<li>a</li>
<li>b</li>
<li>c</li>
</ul>
```
