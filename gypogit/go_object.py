from cffi import FFI
import codecs
import re
from six import text_type
import weakref


class GoGitError(Exception):
    pass


class GoObject(object):
    ffi = FFI()
    lib = None

    def __new__(cls, *args, **kwargs):
        assert cls.lib is not None
        return super(GoObject, cls).__new__(cls, *args)

    @classmethod
    def initialize(cls, header_path, library_path):
        with codecs.open(header_path, "r", "utf-8") as fin:
            src = fin.read()
            src = re.sub("#ifdef.*\n.*\n#endif|#.*|.*_Complex.*|"
                         ".*_check_for_64_bit_pointer_matching_GoInt.*",
                         "", src)
            src = src.replace("__SIZE_TYPE__", "uintptr_t")
            cls.ffi.cdef(src)
        cls.lib = cls.ffi.dlopen(library_path)

    def __init__(self, handle):
        self._handle = handle
        self._strings = weakref.WeakKeyDictionary()

    def __del__(self):
        self.lib.c_dispose(self._handle)

    @property
    def handle(self):
        return self._handle

    @classmethod
    def checked(cls, result):
        smth, err = result
        return smth

    @classmethod
    def string(cls, contents, owner=None):
        """
        Converts Python string to Go string and vice versa.
        :param contents: str, unicode, bytes or FFi.CData struct of GoString
        :param owner: If contents is a Python string, owner of the underlying
        buffer. That buffer must be saved separately if owner is None.
        :return: Py->Go: either GoString or GoString, char* (owner is None)
                 Go->Py: str, unicode
        """
        if isinstance(contents, cls.ffi.CData):
            return cls.ffi.string(contents.p, contents.n).decode("utf-8")
        if isinstance(contents, text_type):
            contents = contents.encode("utf-8")
        char_ptr = cls.ffi.new("char[]", contents)
        go_str = cls.ffi.new("GoString*", {
            "p": char_ptr,
            "n": len(contents)
        })[0]
        if owner is not None:
            owner._strings[go_str] = char_ptr
            return go_str
        else:
            return go_str, char_ptr
