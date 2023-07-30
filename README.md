# fvl
Find Vlan ACL by ip from cisco backup files directory

Example use:
```bash
user@ubuntu$ fvl 172.24.62.2
Host: comm3 Iface: Vlan2621 Vrf:  IfaceIp: 172.24.62.1/29 AclIn:  AclOut: vlan2621_out
user@ubuntu$
user@ubuntu$ fvl 172.24.64.198
Host: comm3 Iface: Vlan2651 Vrf: SMVL IfaceIp: 172.24.64.193/29 AclIn: vlan2651_in AclOut: 
user@ubuntu$ fvl 172.24.64.198 172.24.6.1
Host: comm3 Iface: Vlan2651 Vrf: SMVL IfaceIp: 172.24.64.193/29 AclIn: vlan2651_in AclOut: 
Destination:
Host: comm1 Iface: Vlan2006 Vrf:  IfaceIp: 172.24.6.1/27 AclIn: vlan2006_in AclOut: vlan2006_out
```

Example backup file comm1:
```
...
!
interface Vlan2006
 description new
 ip address 172.24.6.1 255.255.255.224
 ip access-group vlan2006_in in
 ip access-group vlan2006_out out
 no ip redirects
 no ip unreachables
...
```
Example backup file comm3:
```
...
!
interface Vlan2621
 ip address 172.24.62.1 255.255.255.248
 ip access-group vlan2621_out out
 no ip redirects
 no ip unreachables
!
interface Vlan2651
 description SMDC
 vrf forwarding SMVL
 ip address 172.24.64.193 255.255.255.248
 ip access-group vlan2651_in in
 no ip redirects
 no ip unreachables
!
...
```

``
