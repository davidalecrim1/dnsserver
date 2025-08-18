##  Tasks
- [x] Create a new test to see if the DNS questions is working
- [x] Parse a full DNS messages with header and questions
- [x] Move to the answers piece of the code crafters.
- [x] Parse the header section getting data from the request for the flags.
- [x] Parse the questions section.
- [x] Parse a compressed packet (this is no longer needed given the RFC 9619). More on: https://forum.codecrafters.io/t/how-to-get-dig-to-send-multiple-questions-at-once/5087/5
- [x] Forwarding server. 

## Bugs
- [x] There is an error message when the forwarding is not working.
    - Warning: query response not set
    - Warning: Message parser reports malformed message packet.

## Ideas
- [ ] Compression is still valid for answers section on responses.