# nat-pmp-mock
A Go language mock server for the NAT-PMP

# Purpose
This mock server for the NAT-PMP testing. I think this library will be used for cases like this.
- Implemented a NAT-PMP client. I'd like to see if it meets the specs.
- I am trying to import and use an external NAT-PMP library. I want to check if that library meets the spec.

In cases like this, writing test scripts is obscure(e.g. NAT does not support PMP, Scenario test for NAT restart). If you want to test it, you can configure NAT and check whether it works, but it is difficult to automate. As mentioned once, suppose you want to test for a NAT restart, you need to actually shut down the NAT after running the script. Also imagine being behind an unprivileged NAT. You must ask your administrator to turn on the NAT-PMP option. And it's possible that he's a security admin and won't turn that option on without a good reason.
<br>Therefore, this library implements the functions of NAT-Gateway mentioned in RFC-6886, so it can respond correctly to the request of the NAT-PMP Client. By using this mock NAT server, you can easily test various scenarios without manipulating the actual NAT device.

<br>For the specifications that the corresponding mock server meets, refer to the following.


