Name:       another-http-check
Version:    %(date +%%Y%%m%%d)
Release:    1
Summary:    HTTP check for Nagios/Icinga
License:    APACHE 2

%description
HTTP check for Nagios/Icinga

%prep
# pass

%build
# pass

%install
mkdir -p %{buildroot}/usr/lib64/nagios/plugins/
install -m 755 /app/another-http-check %{buildroot}/usr/lib64/nagios/plugins/another-http-check

%files
/usr/lib64/nagios/plugins/another-http-check

%changelog
# pass
