# Fix the URL

## URL from original script

<https://www.shopgoodwill.com/Listings?st=&sg=Ending&c=&s=&lp=0&hp=999999&sbn=false&spo=false&snpo=true&socs=false&sd=false&sca=false&caed=11%2F25%2F2021&cadb=7&scs=false&sis=false&col=1&p=1&ps=40&desc=false&ss=0&UseBuyerPrefs=true>

## URL work in progress

* remove "www"
* change path from Listings to categories/listing
* add "coach" as search term
* add end date as caed
* remove "Ending" as sg (returns error)
* add c=27
* change sbn=false to sbn=
* add &sus=true&cln=2&catIds=10,27&pn=&wc=false&mci=false&hmt=false

<https://shopgoodwill.com/categories/listing?st=coach&sg=&c=27&s=&lp=0&hp=999999&sbn=&spo=false&snpo=true&socs=false&sd=false&sca=false&caed=11%2F25%2F2021&cadb=7&scs=false&sis=false&col=1&p=1&ps=40&desc=false&ss=0&UseBuyerPrefs=true&sus=true&cln=2&catIds=10,27&pn=&wc=false&mci=false&hmt=false>

## URL from new site

<https://shopgoodwill.com/categories/listing?st=coach&sg=&c=27&s=&lp=0&hp=999999&sbn=&spo=false&snpo=true&socs=false&sd=false&sca=false&caed=11%2F25%2F2021&cadb=7&scs=false&sis=false&col=1&p=1&ps=40&desc=false&ss=0&UseBuyerPrefs=true&sus=true&cln=2&catIds=10,27&pn=&wc=false&mci=false&hmt=false>
