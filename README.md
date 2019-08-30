# lessisbetterbot
A quick hack bot to replace a bloated supy

Long story short: we were using supybot only for the url title checker and I got tired of tracebacks because of unicode and similar so, here we are

```
Usage of lessisbetterbot:                                                   
lessisbetterbot [flags] /path/to/config.ini:                                
--createconfig  (= false)                                                       
    create the initial config on the config file, will only work if the file doe
s not exist                                                                     
--network (= "freenode")                                                        
    the name of the url to be connected to (must match config file section)     
```
