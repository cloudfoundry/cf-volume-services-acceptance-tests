echo "VCAP_SERVICES"
echo "###############################################################################"
echo $Env:VCAP_SERVICES
echo "###############################################################################"
echo $Env:VCAP_SERVICES > vcap-services.txt

# look for a credentials block from a user provided service, a secure-credentials service, or a credhub service
$base = echo $Env:VCAP_SERVICES | ./jq -c '.["""user-provided"""]'
if ($base -eq "" -or $base -eq $null -or $base -eq "null") {
  $base = echo $Env:VCAP_SERVICES | ./jq -c '.["""secure-credentials"""]'
}
if ($base -eq "" -or $base -eq $null -or $base -eq "null") {
 $base = echo $Env:VCAP_SERVICES | ./jq -c '.credhub'
}

if ($base -eq "" -or $base -eq $null -or $base -eq "null") {
  echo "Unable to find volume credentials...skipping mount"
} else {
  echo "FINAL-BASE:"
  echo $base
  $smbshare=echo $base | ./jq -r '.[0].credentials.share'
  $smbuser=echo $base | ./jq -r '.[0].credentials.user'
  $smbpassword=echo $base | ./jq -r '.[0].credentials.password'

  echo "USER: $smbuser"
  echo "SHARE: $smbshare"
  echo "password: $smbpassword"

  New-SmbMapping -LocalPath 'Q:' -RemotePath $smbshare -UserName $smbuser -Password $smbpassword

  # inject the path for the new mount back into VCAP_SERVICES
  $Env:VCAP_SERVICES=echo $Env:VCAP_SERVICES | ./jq -r '. + {"smb":[{"volume_mounts":[{"container_dir":"""Q:""","device_type":"""shared""","mode":"""rw"""}]}]}'
}

./server.exe
