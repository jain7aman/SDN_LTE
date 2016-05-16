## SDN EPC User Guide

The manual contains all the steps to setup the SDN-based EPC. The entire setup (see Fig. 1) is primarily composed of the following components:

1. Floodlight controller
2. MySQL database (for HSS)
3. RAN simulator
4. Open vSwitch (for the gateways)
5. Sink program

<!--<img src="https://github.com/jain7aman/SDN_LTE/blob/master/SDN_LTE/images/sdn_epc_arch.png" alt="Fig. 1: SDN-based LTE EPC implementation" width="200" height="200" />-->

We need 6 machines in total to have the entire setup running.

## Requirements
* Ubuntu 14.04 (64-bit)
* RAM: 4 GB

## Controller Machine (MySQL and Floodlight)


