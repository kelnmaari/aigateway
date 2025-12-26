%define name aigateway-repo
# Repository configuration package for AIGateway

Name:           %{name}
Version:        1.0
Release:        1%{?dist}
Summary:        YUM repository configuration for AIGateway
BuildArch:      noarch

License:        MIT
URL:            https://gitlab.alexue4.dev

%description
This package installs the YUM repository configuration for AIGateway
(Ollama OpenAI Proxy). After installation, you can install AIGateway with:

    yum install ollama-openai-proxy

%prep
# Nothing to prepare

%build
# Nothing to build

%install
mkdir -p %{buildroot}%{_sysconfdir}/yum.repos.d
install -m 644 %{_sourcedir}/aigateway.repo %{buildroot}%{_sysconfdir}/yum.repos.d/aigateway.repo

%files
%config(noreplace) %{_sysconfdir}/yum.repos.d/aigateway.repo

%post
echo ""
echo "=============================================="
echo " AIGateway repository configured!"
echo "=============================================="
echo ""
echo " Install AIGateway:"
echo "   yum install ollama-openai-proxy"
echo ""
echo " List available versions:"
echo "   yum list available ollama-openai-proxy"
echo ""

%changelog
* %(date "+%a %b %d %Y") AIGateway Team <team@alexue4.dev> - 1.0-1
- Initial repository configuration package

