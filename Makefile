simple-test-1:
	nc -u localhost 2053

simple-test-2:
	dig @127.0.0.1 -p 2053 example.com

run:
	air .