// Uses the existing SSH credential. Never prints the access code or session token.
import {execFileSync,spawnSync} from 'node:child_process';
const base=process.env.CLUB_TEST_URL||'http://127.0.0.1:18081';
function ssh(command){let error;for(let i=0;i<5;i++){try{return execFileSync('ssh',['-o','ConnectTimeout=10','my-cloud',command],{encoding:'utf8'}).trim()}catch(e){error=e;Atomics.wait(new Int32Array(new SharedArrayBuffer(4)),0,0,1000)}}throw error}
const code=ssh('sed -n "s/^CLUB_ACCESS_CODE=//p" /home/deploy/club-test/service.env');
const login=await fetch(base+'/api/v1/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({code,userId:'owner'})});
if(!login.ok)throw new Error('Remote login failed');
const {token}=await login.json();
const read=async()=>{const r=await fetch(base+'/api/v1/state',{headers:{Authorization:`Bearer ${token}`}});if(!r.ok)throw new Error('Remote state failed');return r.json()};
if(!process.env.CLUB_SKIP_BROWSER){const result=spawnSync('npx',['playwright','test'],{cwd:new URL('../front/',import.meta.url),env:{...process.env,CLUB_TEST_URL:base,CLUB_TEST_CODE:code},stdio:'inherit'});if(result.status!==0)process.exit(result.status||1)}
const before=await read();
ssh('systemctl --user restart club-test.service');
let after;for(let attempt=0;attempt<10;attempt++){try{after=await read();break}catch{await new Promise(r=>setTimeout(r,500))}}
if(!after||JSON.stringify(before.orders)!==JSON.stringify(after.orders))throw new Error('Remote persistence check failed');
console.log(`Remote persistence verified: ${after.orders.length} orders survived service restart.${process.env.CLUB_SKIP_BROWSER?'':' Browser workflow and two-device sync passed.'}`);
