const {test} = require('node:test');
const assert = require('node:assert/strict');
const os = require('node:os');
const installer = require('./install');
test('installer names match published platform assets',()=>{
 const oldPlatform=os.platform,oldArch=os.arch;
 try {
  for(const [platform,arch,name] of [['linux','x64','linux-amd64.tar.gz'],['darwin','arm64','darwin-arm64.tar.gz'],['win32','x64','windows-amd64.zip'],['win32','arm64','windows-arm64.zip']]){
   os.platform=()=>platform;os.arch=()=>arch;assert.equal(installer.getAssetName(),'player2-kanban-'+name);
  }
  os.platform=()=> 'unknown';assert.throws(()=>installer.getAssetName(),/Unsupported/);
 } finally{os.platform=oldPlatform;os.arch=oldArch;}
});
test('redirect guard rejects insecure and looping downloads', async()=>{
 await assert.rejects(installer.followRedirects('http://example.com'),/Invalid/);
 await assert.rejects(installer.followRedirects('https://example.com',6),/Invalid/);
});

test('checksum rejects tampered or missing release assets',()=>{
 const data=Buffer.from('binary');const hash=require('node:crypto').createHash('sha256').update(data).digest('hex');
 assert.doesNotThrow(()=>installer.verifyChecksum(data,hash+'  asset.tar.gz','asset.tar.gz'));
 assert.throws(()=>installer.verifyChecksum(Buffer.from('tampered'),hash+'  asset.tar.gz','asset.tar.gz'),/checksum/);
 assert.throws(()=>installer.verifyChecksum(data,'','asset.tar.gz'),/checksum/);
});
