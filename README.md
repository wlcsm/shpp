# Simple Shell Preprocessor

Executes all the code inside blocks delimited by `%{` `}%` and substitutes it back into the source. 

More specifically, the code inside the blocks will be written to a temporary file with a shebang added and executed. Its output will then be substituted back into the input stream and printed to STDOUT.

Inspired by the rc templating language https://werc.cat-v.org/docs/rc-template-lang.

## Installation

```
go install github.com/wlcsm/shpp@latest
```

## Configuration

The temporary file can be configured with the `SHPP_TMPFILE` (default: `./shpp-cache`) environment variable and the program used to execute the scripts can be configured with `SHPP_PROGRAM` (default: `/bin/sh`) environment variable.

## Examples

```html
# example.txt
<title>Hello from %{printf "the Shell!"}%</title>
```

```
$ shpp example.txt
Hello from the Shell!
```

Environment variables are also passed into the program

```html
# example.txt
Hi %{printf $NAME}%
```

```
$ NAME=user shpp example.txt
Hi user
```

We can also create templates by using a main template file and passing data from stdin. This is a very paired down version of the pipeline used on my website.

```html
# html_template.html
<!DOCTYPE html>
<html lang="en">
  <head></head>
  <body>
    <div id="content">
      %{cat}%
    </div>
  </body>
</html>

# blog_post.html
<h1>New blog!</h1>
<p>Check out my new website!</p>
```

```
$ shpp html_template.html < blog_post.html

# html_template.html
<!DOCTYPE html>
<html lang="en">
  <head></head>
  <body>
    <div id="content">
      <h1>New blog!</h1>
<p>Check out my new website!</p>
    </div>
  </body>
</html>
```
