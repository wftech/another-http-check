#!/bin/sh

test -f another-http-check || exit 1

./another-http-check -H httpbin.org -p 443 -u /status/200 -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /status/201 -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u '/anything?foobar=baz' -s 'foobar' -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u '/anything?foobar=baz' -s 'fuubar' -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /headers -s 'icinga-http-check' -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /basic-auth/user/password \
    --auth-basic -a user:password -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /basic-auth/user/password \
    --auth-basic -a user:password_ -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /basic-auth/user/password \
    -a user:password -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /basic-auth/user/password \
    -a user -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /basic-auth/user/password \
    -a user:password --guess-auth -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -u /delay/10 -t 5 -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -C 15 -S -v
echo "Status code: $?"
echo

./another-http-check -H httpbin.org -p 443 -C 999999,999999 -S -v
echo "Status code: $?"
echo

./another-http-check -H self-signed.badssl.com -p 443 -v
echo "Status code: $?"
echo

./another-http-check -H self-signed.badssl.com -p 443 -k -v
echo "Status code: $?"
