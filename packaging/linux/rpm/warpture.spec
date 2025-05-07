Name:           warpture
Version:        1.0.0
Release:        1%{?dist}
Summary:        Cloudflare WARP GUI Manager with App Split Tunneling
License:        MIT
URL:            https://github.com/mehranredrose/WARPture
BuildArch:      x86_64

Requires:       gtk3, libnotify, nss

%description
WARPture provides a system tray GUI for managing Cloudflare WARP
with per-application split tunneling support.

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/share/applications
mkdir -p %{buildroot}/usr/share/pixmaps
install -m 0755 warpture %{buildroot}/usr/bin/warpture
install -m 0644 warpture.desktop %{buildroot}/usr/share/applications/
install -m 0644 warpture.png %{buildroot}/usr/share/pixmaps/

%files
/usr/bin/warpture
/usr/share/applications/warpture.desktop
/usr/share/pixmaps/warpture.png

%changelog
* Mon Jan 06 2026 mehranredrose <77771629+mehranredrose@users.noreply.github.com> - 1.0.0-1
- Initial release v1.0.0
