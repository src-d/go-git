from cffi import FFI
import codecs
import re
from six import text_type
import weakref


class GoGitError(Exception):
    def __init__(self, code, message):
        super(GoGitError, self).__init__("[%d] %s" % (code, message))
        self.code = code
        self.message = message


class GoObject(object):
    ffi = FFI()
    lib = None
    INVALID_HANDLE = 0

    def __new__(cls, *args, **kwargs):
        assert cls.lib is not None
        return object.__new__(cls)

    @classmethod
    def initialize_go(cls, header_path, library_path):
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

    def __repr__(self):
        general_str = super(GoObject, self).__repr__()
        return "%s | 0x%x>" % (general_str[:-1], self._handle)

    def __hash__(self):
        return self._handle

    @property
    def handle(self):
        return self._handle

    @classmethod
    def _checked(cls, result):
        unpacked = tuple(getattr(result, f[0])
                         for f in cls.ffi.typeof(result).fields)
        if cls.ffi.typeof(unpacked[-1]).cname == "GoString":
            assert len(unpacked) >= 2
            assert isinstance(unpacked[-2], int)
            if unpacked[-2] == 0:
                if len(unpacked) == 2:
                    return
                if len(unpacked) == 3:
                    return unpacked[0]
                return unpacked[:-2]
            raise GoGitError(unpacked[-2], cls._string(unpacked[-1]))
        assert isinstance(unpacked[-1], int)
        if unpacked[-1] == 0:
            if len(unpacked) == 1:
                return
            if len(unpacked) == 2:
                return unpacked[-1]
            return unpacked[:-1]
        raise GoGitError(unpacked[-1], "<no message>")

    @classmethod
    def _string(cls, contents, owner=None):
        """
        Converts Python string to Go string and vice versa.
        :param contents: str, unicode, bytes or FFi.CData struct of GoString
        :param owner: If contents is a Python string, owner of the underlying
        buffer. That buffer must be saved separately if owner is None.
        :return: Py->Go: either GoString or GoString, char* (owner is None)
                 Go->Py: str, unicode
        """
        if isinstance(contents, cls.ffi.CData):
            if contents.n == 0:
                return ""
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

    @classmethod
    def dump_go(cls):
        cls.lib.c_dump_objects()
