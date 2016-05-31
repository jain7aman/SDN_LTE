## SDN-based LTE EPC

This is a SDN-based implementation of the LTE EPC (Evolved Packet Core). Following the design principles of SDN (Software Defined Networking), we have identified and separated out the control and data plane functionalities of the EPC components. We have implemented the basic functionalities including the `Attach` and `Detach` procedures. Although there have been several proposals in this field, but currently there does not exist any open-source framework for research and experimentation (to the best of our knowledge). This project can be used by researchers to compare and analyse different design choices on the basis of various performance metrics. Besides, new functionalities can be developed on top of the existing code corresponding to any new specifications.

This is a *Beta version* and we are in the process of incorporating more procedures, thus making it more standards compliant. We expect that this project will encourage more research and innovation in this space.

#### Contents ####

- Source code for various EPC components (`Controller`, `RAN` and `EPC Gateways`)
- A [user guide](README_User.md) containing the setup and installation instructions.
- A [developer guide](README_Developer.md) which explains the structure of the source code.

#### Authors ####

* [Aman Jain](https://www.linkedin.com/in/aman-jain-04590515)
* [Sunny Kumar Lohani](https://www.linkedin.com/in/sunny-lohani-a52a958b)
* [Prof. Mythili Vutukuru](https://www.cse.iitb.ac.in/~mythili/)
