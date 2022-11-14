# Simple preprocessor

Just runs `sh` on all the code inside %{ }% blocks in the source. Inspired by the rc templating languages https://werc.cat-v.org/docs/rc-template-lang

Everything is passed through stdin.
May pass environment variables

## Examples

```html
<ul>
%{
for(i in a b c) {
   echo '<li>'$i'</li>'
}
}%
</uL>
```

becomes

```html
<ul>
<li>a</li>
<li>b</li>
<li>c</li>
</ul>
```

Use envirnoment variables to pass data to the scripts

```html
# index.html
<title>%{echo $NAME}%</title>
```

run with

```
NAME=myname shpp < index.html
```

produces

```html
<title>myname</title>
```
