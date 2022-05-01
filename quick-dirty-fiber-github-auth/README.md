# Fiber + GitHub Auth

I wanted to make a quick POC for authenticating with GitHub via a Fiber application.
It turned out to be anything but quick.

I spent a lot of time mucking with [goth](https://github.com/markbates/goth) and also [gologin](https://github.com/dghubble/gologin). While they both worked for getting the actual OAuth flow to work, using them with Fiber's session wasn't all that nice. At the state it's at, I'm okay with it as a basic example of how it could be done but I think the approach could be better.

