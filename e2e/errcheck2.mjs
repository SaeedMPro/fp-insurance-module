import puppeteer from 'puppeteer-core'
const OUT='/tmp/claude-1000/-home-saeedm-Documents-fp/120c0151-aecc-45c3-8f75-55f4fdb5dae4/scratchpad/devup'
const wait=ms=>new Promise(r=>setTimeout(r,ms))
const b=await puppeteer.launch({executablePath:'/usr/bin/google-chrome-stable',headless:'new',args:['--no-sandbox','--window-size=1200,850']})
const p=await b.newPage(); await p.setViewport({width:1200,height:850})
await p.emulateMediaFeatures([{name:'prefers-color-scheme',value:'light'}])

// 1) wrong password -> Persian message from the map
await p.goto('http://localhost:5173/login',{waitUntil:'networkidle2'})
await p.type('#username','admin'); await p.type('#password','wrong-password')
await p.click('button[type=submit]'); await wait(1500)
console.log('۱) گذرواژه غلط →', await p.evaluate(()=>{
  const el=[...document.querySelectorAll('div')].find(d=>/نامعتبر|invalid/i.test(d.textContent)&&d.children.length===0)
  return el?el.textContent.trim():'(پیدا نشد)'}))
await p.screenshot({path:`${OUT}/4-persian-error-login.png`})

// 2) employee hitting an admin-only page via URL -> guard redirect (no English leak)
await p.evaluate(()=>{document.querySelector('#password').value=''})
await p.goto('http://localhost:5173/login',{waitUntil:'networkidle2'})
await p.type('#username','sara.ahmadi'); await p.type('#password','Employee123!')
await Promise.all([p.waitForNavigation({waitUntil:'networkidle2'}).catch(()=>{}),p.click('button[type=submit]')]); await wait(900)
await p.goto('http://localhost:5173/users',{waitUntil:'networkidle2'}); await wait(900)
console.log('۲) کارمند → /users ریدایرکت شد به:', p.url())

// 3) direct check of the translator over every backend message we know
const res=await p.evaluate(async ()=>{
  const out=[]
  const tries=[['sara.ahmadi','Employee123!','/api/v1/coverage-rules','POST',{}]]
  for(const [u,pw,path,method,body] of tries){
    const r=await fetch('http://localhost:18080/api/v1/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:pw})})
    const {token}=await r.json()
    const r2=await fetch('http://localhost:18080'+path,{method,headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify(body)})
    out.push((await r2.json()).error)
  }
  return out
})
console.log('۳) پیام خام API (انگلیسی، درست است):', res[0])
await b.close()
