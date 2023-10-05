for-each
================
is a simple command line tool to execute a command or a list of commands on a list of servers.
You can provide the list of servers either as a file or via stdin.   
The list must be formatted as a table, with the server name in the first column and 
the server IP in the second column.  
The tool does not parse the rest of the table, which can contain any data convenient for filtering the list. 

The tool utilizes the system's SSH utility and does not request passwords from the tty. 
Therefore, key-based authentication is required for the tool 
(as well as membership in the 'wheel' group for passwordless sudo, if needed).

The tool automatically creates log files for each server in the ./logs folder 
unless the --no-logs option is specified.   
Each log file created by the tool contains all executed commands, 
along with timestamps and the output for each command.

Examples:
-----------------

- Execute ***cat /etc/hostname*** on a list of servers, piping the list from servers from stdin:

      cat <<END | for-each cat /etc/hostname  
      server1 192.168.0.10  
      server2 192.168.0.11   
      END  

- Execute ***cat /etc/hostname*** on a list of servers, specifying the list of servers as a file:

      cat > servers.txt <<END  
      server1 192.168.0.10 =gpu  
      server2 192.168.0.11 =worker  
      server3 192.168.0.12 =gpu  
      END  

      for-each -f servers.txt cat /etc/hostname 

- Execute ***cat /etc/hostname*** on gpu servers only:

      cat > servers.txt <<END  
      server1 192.168.0.10 =gpu  
      server2 192.168.0.11 =worker  
      server3 192.168.0.12 =gpu  
      END  

      grep =gpu servers.txt | for-each cat /etc/hostname
- Execute a list of commands on a list of servers.  
I intentionally don't call it a script, because the tool does not really run it as a remote script.  
Instead it executes the commands from the list via ssh one by one.  
This way it does not create any files on the remote servers.

      cat > commands.txt <<END  
      apt update  
      apt install -y   
      END
    
      cat <<END | for-each -s commands.txt  
      server1 192.168.0.10  
      server2 192.168.0.11   
      END
  
