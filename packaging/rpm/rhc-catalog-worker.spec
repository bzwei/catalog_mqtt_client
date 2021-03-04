%define debug_package %{nil}
%global commitid f31877eb50e732f2f0accb14451bf877148c44bb

Name:    rhc-catalog-worker
Version: 0.1.1
Release: 1%{?dist}
Epoch:   1
Summary: Catalog Worker for RHC Daemon links Ansible Tower to cloud.redhat.com
License: ASL 2.0
URL:     https://github.com/RedHatInsights/rhc-catalog-worker

Source0: %{url}/archive/v%{version}.tar.gz

ExclusiveArch: %{go_arches}

BuildRequires: golang

%description
%{name} runs as a worker under rhcd, waits for requests from cloud.redhat.com
and makes REST API calls to Ansible Tower.

%prep
%autosetup

%build
make VERSION=%{version} SHA=%{commitid}

%install
mkdir -p %{buildroot}/{etc/rhc/workers,usr/libexec/rhc}
%{__install} -m 755 %{_builddir}/%{name}-%{version}/%{name} %{buildroot}/usr/libexec/rhc

%files
%{_libexecdir}/rhc/%{name}

%changelog
