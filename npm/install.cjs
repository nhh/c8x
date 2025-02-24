const fs = require('fs');
const os = require('os');

const c8x = JSON.parse(fs.readFileSync("package.json").toString())

const version = c8x.version
const platform = os.platform()
const binaryName = platform === "win32" ? `c8x.exe` : `c8x`
const downloadUrl = `https://github.com/nhh/c8x/releases/download/v${version}-alpha/c8x-${platform}-x86_64${platform === "win32" ? ".exe" : ""}`

fetch(downloadUrl)
    .then(res => res.status === 200 ? res : Promise.reject(`Download failed: ${res.status}/ ${res.statusText}`))
    .then(res => res.arrayBuffer())
    .then(bytes => fs.writeFileSync(binaryName, new Uint8Array(bytes)))
    .catch(e => console.error(e));

if(platform !== "win32") {
    fs.chmodSync(binaryName, fs.constants.S_IRWXU)
}
