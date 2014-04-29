import os, re

def purge(dir, pattern):
    for f in os.listdir(dir):
        print f
        if(re.search(pattern, f)):
            os.remove(os.path.join(dir, f))


purge(os.getcwd(), "8080\||9090\||8081\||8082\||8083\||9990\|")
