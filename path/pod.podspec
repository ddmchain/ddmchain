Pod::Spec.new do |spec|
  spec.name         = 'Gddm'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/ddmchain/go-ddmchain'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS DDMchain Client'
  spec.source       = { :git => 'https://github.com/ddmchain/go-ddmchain.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gddm.framework'

	spec.prepare_command = <<-CMD
    curl https://gddmstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gddm.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
