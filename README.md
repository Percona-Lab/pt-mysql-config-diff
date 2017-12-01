# MySQL Config Diff

This program can compare configurations and default values from different MySQL config sources like cnf files, SHOW VARIABLES and MySQL defaults parsed from `mysqld` help.  

## Usage

```
pt-mysql-config-diff [--format=text/json] <src_1> <src_2>
```

where `src` could be a file name pointing to a `.cnf` file or to a file having MySQL default values from `mysqld` help or a dsn in the form of a default pt-tool dsn parameter: `h=<host>,P=<port>,u=<user>,p=<password>`.

## Usage examples
### Comparing .cnf vs .cnf files.
When comparing 2 `cnf` files, the program will show all keys having differences between the 2 files, including missing keys in both files.

```
pt-mysql-config-diff file1.cnf file2.cnf
```

**Example:**

`cnf1.cnf`
```
[mysqld]
key1=value1
key2= 2
key3= true
```

`cnf2.cnf`
```
[mysqld]
key1=value1
key2=3
key4=true
```

`./pt-mysql-config-diff --format=text cnf1.cnf cnf2.cnf`
```
key2:         2 <->         3
key3:      true <-> <Missing>
key4: <Missing> <->      true
```

### Comparing .cnf vs SHOW VARIABLES

When comparing `.cnf` files vs `SHOW VARIABLES`, only configuration variables that exist in the cnf file are compared.  

**Example: comparing a cnf vs a MySQL 5.7 instance**  

`cnf3.cnf`  
```
[mysqld]
innodb_buffer_pool_size=512M
log_slow_rate_limit=100.1234
log_slow_verbosity=full
```

`./pt-mysql-config-diff --format=text ~/cnf3.cnf h=127.1,P=3306,u=root`  
```
     log_slow_verbosity:      full <-> <Missing>
innodb_buffer_pool_size: 536870912 <-> 134217728
    log_slow_rate_limit:  100.1234 <-> <Missing>
```

### Comparing SHOW VARIABLES vs SHOW VARIABLES

When comparing the same type of configs (SHOW VARIABLES on both sides), the program will show all keys having differences between the 2 instances, including missing keys in both sides.  

**Example: Comparing MySQL 5.7 vs MySQL 5.6**  
*Note: the output has been truncated and only a few values are here as an example*  

Having MySQL 5.7 on port `3306` and MySQL 5.6 on port `3308`:  

`./pt-mysql-config-diff --format=text h=127.1,P=3306,u=root h=127.1,P=3308,u=root`  

```

                                          innodb_version:                                5.7.20 <->                               5.6.38
                      innodb_buffer_pool_load_at_startup:                                     1 <->                                    0
                                    session_track_schema:                                    ON <->                            <Missing>
                                                ssl_cert:                       server-cert.pem <->                                     
                    performance_schema_setup_actors_size:                                    -1 <->                                  100
                                           timed_mutexes:                             <Missing> <->                                  OFF
                  performance_schema_max_mutex_instances:                                    -1 <->                                15906
                                   innodb_undo_directory:                                    ./ <->                                    .
                         simplified_binlog_gtid_recovery:                             <Missing> <->                                  OFF
                                     session_track_gtids:                                   OFF <->                            <Missing>
                             sha256_password_proxy_users:                                   OFF <->                            <Missing>
                                           rbr_exec_mode:                                STRICT <->                            <Missing>
                    performance_schema_max_table_handles:                                    -1 <->                                 4000
                   performance_schema_max_metadata_locks:                                    -1 <->                            <Missing>
                        log_statements_unsafe_for_binlog:                                    ON <->                            <Missing>

```

### Getting the list of variables having non-default values

To achieve this, you first need to generate a list of defaults for the MySQL version you are running:  

`touch /tmp/my.cnf`  
`<path-to-mysql-bin>/mysqld --verbose --defaults-file=/tmp/my.cnf --help > ~/my-5.7.defaults`

and then you can compare the values from `SHOW VARIABLES` against the defaults:  

`pt-mysql-config-diff --format=text h=127.1,P=3306,u=root ~/my-5.7.defaults`  

```
                                                   daemonize:                            <Missing> <->                               FALSE
                                                   federated:                            <Missing> <->                                  ON
                                            general_log_file:      /var/lib/mysql/fa5f51a13d1a.log <-> /usr/local/mysql/data/karl-OMEN.log
                                                log_warnings:                                    2 <->                                   0
       performance_schema_consumer_events_waits_history_long:                            <Missing> <->                               FALSE
                                           skip_grant_tables:                            <Missing> <->                               FALSE
                                                   blackhole:                            <Missing> <->                                  ON
                                                      socket:          /var/run/mysqld/mysqld.sock <->                     /tmp/mysql.sock
                                            open_files_limit:                              1048576 <->                                5000
                             performance_schema_digests_size:                                10000 <->                                  -1
                                                 log_tc_size:                            <Missing> <->                               24576
                                           myisam_block_size:                            <Missing> <->                                1024
                                                     datadir:                      /var/lib/mysql/ <->              /usr/local/mysql/data/
       performance_schema_consumer_events_statements_history:                            <Missing> <->                                TRUE
                                            log_short_format:                            <Missing> <->                               FALSE
```


### TODO
- [ ] Add option to skip missing values on right/left side
- [ ] Add option to skip certain variables
- [ ] Add option to show big numbers in human readable format (1K, 1M)