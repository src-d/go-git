from .go_object import GoObject
from .common import HTTPAuthMethod, SSHPasswordMethod, SSHPublicKeysMethod, \
    Signer
from .repository import Repository
from .remote import Remote

DefaultRemoteName = "origin"

try:
    from .go_paths import HEADER, LIBRARY
    import os

    def makeabs(path):
        if not os.path.isabs(path):
            path = os.path.join(os.path.dirname(__file__), path)
        return path

    GoObject.initialize_go(makeabs(HEADER), makeabs(LIBRARY))
    del HEADER
    del LIBRARY
except (ImportError, IOError):
    pass
