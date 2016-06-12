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
    registry = weakref.WeakValueDictionary()

    def __new__(cls, handle):
        assert cls.lib is not None
        instance = GoObject.registry.get(handle)
        if instance is not None:
            return instance
        return object.__new__(cls)

    @staticmethod
    def initialize_go(header_path, library_path, force=False):
        if GoObject.lib is not None and not force:
            return
        with codecs.open(header_path, "r", "utf-8") as fin:
            src = fin.read()
            src = re.sub("#ifdef.*\n.*\n#endif|#.*|.*_Complex.*|"
                         ".*_check_for_64_bit_pointer_matching_GoInt.*",
                         "", src)
            src = "extern free(void *ptr);\n" + src.replace(
                "__SIZE_TYPE__", "uintptr_t")
            GoObject.ffi.cdef(src)
        GoObject.lib = GoObject.ffi.dlopen(library_path)

    def __init__(self, handle):
        if handle <= self.INVALID_HANDLE:
            raise ValueError("Invalid handle")
        self._handle = handle
        self._deps = weakref.WeakKeyDictionary()
        GoObject.registry[handle] = self

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
        if cls.ffi.typeof(unpacked[-1]).cname == "char *":
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
        Converts Python string to GoString and <char *> to Python string.
        :param contents: str, unicode, bytes or or FFI.CData with <char *>
        :param owner: (Py->Go) instance which owns of the underlying C buffer.
        That buffer must be saved separately if owner is None.
        :return: Py->Go: either GoString or GoString, char* (owner is None)
                 Go->Py: str or unicode
        """
        if isinstance(contents, cls.ffi.CData):
            s = cls.ffi.string(contents).decode("utf-8")
            cls.lib.free(contents)
            return s
        if isinstance(contents, text_type):
            contents = contents.encode("utf-8")
        char_ptr = cls.ffi.new("char[]", contents)
        go_str = cls.ffi.new("GoString*", {
            "p": char_ptr,
            "n": len(contents)
        })[0]
        if owner is not None:
            owner._deps[go_str] = char_ptr
            return go_str
        else:
            return go_str, char_ptr

    @classmethod
    def _bytes(cls, data, owner=None, size=None):
        if isinstance(data, cls.ffi.CData):
            s = cls.ffi.unpack(data, size)
            cls.lib.free(data)
            return s
        assert isinstance(data, (bytes, bytearray, memoryview))
        char_data = cls.ffi.new("char[]", data)
        go_data = cls.ffi.new("GoSlice*", {
            "data": char_data,
            "len": len(data),
            "cap": len(data)
        })[0]
        if owner is not None:
            owner._deps[go_data] = char_data
            return go_data
        else:
            return go_data, char_data

    @classmethod
    def _string_slice(cls, slice, owner=None):
        if isinstance(slice, cls.ffi.CData):
            sarr = cls.ffi.string(slice)
            cls.lib.free(slice)
            return [s.decode("utf-8") for s in sarr.split(b"\xff")]
        raise NotImplementedError()

    @classmethod
    def dump_go(cls):
        cls.lib.c_dump_objects()
