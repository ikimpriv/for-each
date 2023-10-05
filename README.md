for-each
================
is a simple command line tool to send a command or a list of commands to a list of servers.
The list of servers can be provided as a file or from stdin and has to be represented by 
a table with the server name in the first column and the server IP in the second column. The rest of the table is not
parsed and can contain whatever is convenient for filtering the list. 

The tool does not provide any embedded authentication methods, it uses the system ssh tool and 
does not request the password from the tty, so the key-based authentication is required 
(as well as the wheel group for no-password-sudo if needed).

The tool creates log files for each server in the ./logs folder (created automatically) until --no-logs option is specified.  
Each log file contains all executed commands with timestamps and the command output.  

Examples:
-----------------

- Execute ***cat /etc/hostname*** on a list of servers, piping the list from servers from stdin:
    > cat <<END | for-each cat /etc/hostname  
      server1 192.168.0.10  
      server2 192.168.0.11   
      END  

- Execute ***cat /etc/hostname*** on a list of servers, specifying the list of servers as a file:
  > cat > servers.txt <<END  
    server1 192.168.0.10 =gpu  
    server2 192.168.0.11 =worker  
    server3 192.168.0.12 =gpu  
    END  

  > for-each -f servers.txt cat /etc/hostname 

- Execute ***cat /etc/hostname*** on gpu servers only:
    > cat > servers.txt <<END  
      server1 192.168.0.10 =gpu  
      server2 192.168.0.11 =worker  
      server3 192.168.0.12 =gpu  
      END  

    > grep =gpu servers.txt | for-each cat /etc/hostname
- Execute a list of commands on a list of servers.  
I intentionally don't call it a script, because the tool does not really run it as a remote script.  
Instead it executes the commands from the list via ssh one by one.  
This way it does not create any files on the remote servers.
  > cat > commands.txt <<END  
    apt update  
    apt install -y   
    END
    
  > cat <<END | for-each -s commands.txt  
    server1 192.168.0.10  
    server2 192.168.0.11   
    END
  
