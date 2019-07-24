$smbshare=echo $Env:VCAP_SERVICES | ./jq -r '.["""user-provided"""][0].credentials.share'
$smbuser=echo $Env:VCAP_SERVICES | ./jq -r '.["""user-provided"""][0].credentials.user'
$smbpassword=echo $Env:VCAP_SERVICES | ./jq -r '.["""user-provided"""][0].credentials.password'

New-SmbMapping -LocalPath 'Q:' -RemotePath $smbshare -UserName $smbuser -Password $smbpassword

$Env:VCAP_SERVICES=echo $Env:VCAP_SERVICES | ./jq -r '. + {"smb":[{"volume_mounts":[{"container_dir":"""Q:""","device_type":"""shared""","mode":"""rw"""}]}]}'

./server.exe
