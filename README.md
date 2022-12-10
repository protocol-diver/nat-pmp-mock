# nat-pmp-mock
A Go language mock server for the NAT-PMP

# Purpose
This mock server for the NAT-PMP testing. I think this library will be used for cases like this.
- Implemented a NAT-PMP client. I'd like to see if it meets the specs.
- I am trying to import and use an external NAT-PMP library. I want to check if that library meets the spec.

In cases like this, writing test scripts is obscure(e.g. NAT does not support PMP, Scenario test for NAT restart). If you want to test it, you can configure NAT and check whether it works, but it is difficult to automate. Therefore, this library implements the functions of NAT-Gateway mentioned in RFC-6886, and it has been implemented so that it can respond correctly to the request of the NAT-PMP Client.<br>
<br>For the specifications that the corresponding mock server meets, refer to the following.


