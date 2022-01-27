# Milestone 3 - p2p communication

add p2p comms mechanisms to prevent validator deanonymization

#### middleware behavior

- [ ] middleware gossips signed block + initial payload header over p2p

#### required client modifications

- consensus client must implement [New Gossipsub Topics](https://hackmd.io/@paulhauner/H1XifIQ_t#Change-3-New-Gossipsub-Topics)