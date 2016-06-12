gypogit
=======

gypogit is a wrapper for the awesome go-git library working with Python 2.7/3.3+.

Installation
------------
To install gypogit, you must have a working Go compiler.
```
pip install git+https://github.com/src-d/go-git
```

Usage
-----
```py
from gypogit import Repository
r = Repository.New("https://github.com/src-d/go-git")
r.PullDefault()
for c in r.Commits():
    print(c)
```
The naming and classes are left intact to match go-git API.

License
-------
MIT.