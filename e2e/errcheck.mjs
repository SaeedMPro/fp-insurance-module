import puppeteer from 'puppeteer-core'
const OUT='/tmp/claude-1000/-home-saeedm-Documents-fp/120c0151-aecc-45c3-8f75-55f4fdb5dae4/scratchpad/devup'
const wait=ms=>new Promise(r=>setTimeout(r,ms))
const b=await puppeteer.launch({executablePath:'/usr/bin/google-chrome-stable',headless:'new',args:['--no-sandbox','--window-size=1440,1000']})
const p=await b.newPage(); await p.setViewport({width:1440,height:1000})
await p.emulateMediaFeatures([{name:'prefers-color-scheme',value:'light'}])
await p.goto('http://localhost:5173/login',{waitUntil:'networkidle2'})
await p.type('#username','reviewer'); await p.type('#password','Reviewer123!')
await Promise.all([p.waitForNavigation({waitUntil:'networkidle2'}).catch(()=>{}),p.click('button[type=submit]')]); await wait(1000)
// open the first submitted claim and reject with an EMPTY reason -> must show Persian error
const link=await p.$('a[href^="/claims/"]')
await Promise.all([p.waitForNavigation({waitUntil:'networkidle2'}).catch(()=>{}),link.click()]); await wait(900)
p.on('dialog', async d=>{ await d.accept('') })   // empty reason
const btns=await p.evaluate(()=>[...document.querySelectorAll('button')].map(b=>b.textContent.trim()))
console.log('buttons:', btns.filter(Boolean).join(' | '))
// start review first if needed
const startBtn=await p.evaluateHandle(()=>[...document.querySelectorAll('button')].find(b=>b.textContent.trim()==='شروع بررسی')||null)
if (startBtn && await startBtn.evaluate(e=>!!e)) { await startBtn.asElement().click(); await wait(1200) }
const rejectBtn=await p.evaluateHandle(()=>[...document.querySelectorAll('button')].find(b=>b.textContent.trim()==='رد')||null)
if (rejectBtn && await rejectBtn.evaluate(e=>!!e)) { await rejectBtn.asElement().click(); await wait(1500) }
const shown=await p.evaluate(()=>document.body.innerText)
const line=shown.split('\n').find(l=>/دلیل|الزامی|error|not allowed|reason/i.test(l))
console.log('پیام نمایش‌داده‌شده:', line||'(چیزی پیدا نشد)')
await p.screenshot({path:`${OUT}/4-persian-error.png`})
await b.close()
