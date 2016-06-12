from six import string_types

from .go_object import GoObject


class AuthMethod(GoObject):
    pass


class HTTPAuthMethod(AuthMethod):
    @classmethod
    def New(cls, username, password):
        assert isinstance(username, string_types)
        assert isinstance(password, string_types)
        go_username, c_username = cls._string(username)
        go_password, c_password = cls._string(password)
        handle = cls.lib.c_NewBasicAuth(go_username, go_password)
        am = HTTPAuthMethod(handle)
        am._deps[go_username] = c_username
        am._deps[go_password] = c_password
        return am


class SSHPasswordMethod(AuthMethod):
    @classmethod
    def New(cls, username, password):
        assert isinstance(username, string_types)
        assert isinstance(password, string_types)
        go_username, c_username = cls._string(username)
        go_password, c_password = cls._string(password)
        handle = cls.lib.c_ssh_Password_New(go_username, go_password)
        am = SSHPasswordMethod(handle)
        am._deps[go_username] = c_username
        am._deps[go_password] = c_password
        return am

    @property
    def User(self):
        return self._string(self.lib.c_ssh_Password_get_User(self.handle))

    @User.setter
    def User(self, value):
        self.lib.c_ssh_Password_set_User(
            self.handle, self._string(value, self))

    @property
    def Pass(self):
        return self._string(self.lib.c_ssh_Password_get_Pass(self.handle))

    @Pass.setter
    def Pass(self, value):
        self.lib.c_ssh_Password_set_Pass(
            self.handle, self._string(value, self))


class Signer(GoObject):
    @classmethod
    def Parse(cls, data, raw=True):
        if hasattr(data, "read"):
            data = data.read()
        go_data, c_data = cls._bytes(data)
        if raw:
            s = Signer(cls._checked(cls.lib.c_ParseRawPrivateKey(go_data)))
        else:
            s = Signer(cls._checked(cls.lib.c_ParsePrivateKey(go_data)))
        s._deps[go_data] = c_data
        return s


class SSHPublicKeysMethod(AuthMethod):
    @classmethod
    def New(cls, user, signer):
        assert isinstance(user, string_types)
        assert isinstance(signer, Signer)
        go_user, c_user = cls._string(user)
        handle = cls.lib.c_ssh_Password_New(go_user, signer.handle)
        am = SSHPublicKeysMethod(handle)
        am._deps[go_user] = c_user
        return am

    @property
    def User(self):
        return self._string(self.lib.c_ssh_PublicKeys_get_User(self.handle))

    @User.setter
    def User(self, value):
        self.lib.c_ssh_PublicKeys_set_User(
            self.handle, self._string(value, self))

    @property
    def Signer(self):
        return Signer(self.lib.c_ssh_PublicKeys_get_Signer(self.handle))

    @Signer.setter
    def Signer(self, value):
        assert isinstance(value, Signer)
        self.lib.c_ssh_PublicKeys_set_Signer(self.handle, value.handle)


class Signature(GoObject):
    pass
