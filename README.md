# nat-pmp-mock
A Go language mock server for the NAT-PMP

# Purpose
This mock server for the NAT-PMP testing. I think this library will be used for cases like this.
- Implemented a NAT-PMP client. I'd like to see if it meets the specs.
- I am trying to import and use an external NAT-PMP library. I want to check if that library meets the spec.

In cases like this, writing test scripts is obscure(e.g. NAT does not support PMP, Scenario test for NAT restart). If you want to test it, you can configure NAT and check whether it works, but it is difficult to automate. As mentioned once, suppose you want to test for a NAT restart, you need to actually shut down the NAT after running the script. Also imagine being behind an unprivileged NAT. You must ask your administrator to turn on the NAT-PMP option. And it's possible that he's a security admin and won't turn that option on without a good reason.
<br>Therefore, this library implements the functions of NAT-Gateway mentioned in RFC-6886, so it can respond correctly to the request of the NAT-PMP Client. By using this mock NAT server, you can easily test various scenarios without manipulating the actual NAT device.

<br>For the specifications that the corresponding mock server meets, refer to the following.
- [x] Implementing this version of the protocol, receiving a request with a version number other than 0, MUST return result code 1 (Unsupported Version), indicating the highest version number it does support (i.e., 0) in the version field of the response.
- [x] MUST fill in the Seconds Since Start of Epoch field with the time elapsed since its port mapping table was initialized on startup, or reset for any other reason.
- [x] If the result code is non-zero, the value of the External IPv4 Address field is undefined (MUST be set to zero on transmission, and MUST be ignored on reception).
- [ ] MUST send a gratuitous response to the link-local multicast address 224.0.0.1, port 5350, with the packet format above, to notify clients of the external IPv4 address and Seconds Since Start of Epoch.
- [x] The Seconds Since Start of Epoch field in each transmission MUST be updated appropriately to reflect the passage of time, so as not to trigger unnecessary additional mapping renewals.
- [x] Implements this protocol MUST be able to create TCP-only and UDP-only port mappings.
- [ ] MUST NOT automatically create mappings for TCP when the client requests UDP, and vice versa, the NAT gateway MUST reserve the companion port so the same client can choose to map it in the future. For example, if a client requests to map TCP port 80, as long as the client maintains the lease for that TCP port mapping, another client with a different internal IP address MUST NOT be able to successfully acquire the mapping for UDP port 80.
- [x] MUST return an available external port if possible, or return an error code if no external ports are available.
- [x] MUST NOT accept mapping requests destined to the NAT gateway's external IP address or received on its external network interface.
- [x] When a mapping is destroyed successfully as a result of the client explicitly requesting the deletion, the NAT gateway MUST send a response packet that is formatted as defined "Requesting a Mapping". The response MUST contain a result code of 0, the internal port as indicated in the deletion request, an external port of 0, and a lifetime of 0. 
- [x] The NAT gateway MUST respond to a request to destroy a mapping that does not exist as if the request were successful.  This is because of the case where the acknowledgment is lost, and the client retransmits its request to delete the mapping.  In this case, the second request to delete the mapping MUST return the same response packet as the first request.
- [ ] If the client attempts to delete a port mapping that was manually assigned by some kind of configuration tool, the NAT gateway MUST respond with a "Not Authorized" error, result code 2. After receiving such a deletion request, the gateway MUST delete all its UDP or TCP port mappings (depending on the opcode).
- [ ] If the gateway is unable to delete a port mapping, for example, because the mapping was manually configured by the administrator, the gateway MUST still delete as many port mappings as possible, but respond with a non-zero result code.  The exact result code to return depends on the cause of the failure.  If the gateway is able to successfully delete all port mappings as requested, it MUST respond with a result code of zero.
- [x] If the version in the request is not zero, then the NAT-PMP server MUST return the following "Unsupported Version" error response to the client
- [x] If the opcode in the request is 128 or greater, then this is not a request; it's a response, and the NAT-PMP server MUST silently ignore it.
- [x] If the opcode in the request is less than 128, but is not a supported opcode (currently 0, 1, or 2), then the entire request MUST be returned to the sender, with the top bit of the opcode set (to indicate that this is a response) and the result code set to 5 (Unsupported opcode).
- [ ] For version 0 and a supported opcode (0, 1, or 2), if the operation fails for some reason (Not Authorized, Network Failure, or Out of resources), then a valid response MUST be sent to the client, with the top bit of the opcode set (to indicate that this is a response) and the result code set appropriately.  Other fields in the response MUST be set appropriately.
- [x]  If the NAT gateway resets or loses the state of its port mapping table, due to reboot, power failure, or any other reason, it MUST reset its epoch time and begin counting SSSoE from zero again. 
- [ ] When the NAT gateway powers on or clears its port mapping state as the result of a configuration change, it MUST reset the epoch time and re-announce its IPv4 address.
- [x] A network device that is capable of NAT (and NAT-PMP) but is currently configured not to perform that function (e.g., it is acting as a traditional IP router, forwarding packets without modifying them) MUST NOT respond to NAT-PMP requests from clients nor send spontaneous NAT-PMP address-change announcements.
- [ ] All NAT gateways MUST ensure that mappings, however created, are bidirectional.
- [ ] Upon boot, acquisition of an external IPv4 address, subsequent change of the external IPv4 address, reboot, or any other event that may indicate possible loss or change of NAT mapping state, the NAT gateway MUST send a gratuitous response to the link-local multicast address 224.0.0.1, port 5350, with the packet format above, to notify clients of the external IPv4 address and Seconds Since Start of Epoch.
