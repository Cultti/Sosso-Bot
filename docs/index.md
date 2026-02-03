# Sosso-Bot - Discord-botti CS2-liigalle

Tervetuloa Sosso-Botin dokumentaatioon! Tämä botti on suunniteltu helpottamaan Pappaliigan CS2-otteluiden hallintaa Discord-palvelimella.

## Ominaisuudet

- 🎮 **Otteluilmoitukset**: Automaattiset ilmoitukset Faceit-otteluista
- 📅 **Pelipäivä-äänestykset**: Luo äänestyksiä viikon pelipäivistä
- 🏃 **Harkkailmoitukset**: Järjestä harjoituspelejä
- 📢 **Tilaukset**: Hallinnoi kanavien ottelu-ilmoitustilauksia
- 🔔 **Webhook-tuki**: Vastaanota reaaliaikaisia päivityksiä Faceitista

## Pika-aloitus

### Vaatimukset

- Discord-palvelin, jossa sinulla on admin-oikeudet
- Botin kutsulinkki (pyydä järjestelmänvalvojalta)

### Botin lisääminen palvelimelle

1. Käytä järjestelmänvalvojalta saamaasi kutsulinkiä
2. Valitse palvelin, jolle haluat botin lisätä
3. Hyväksy tarvittavat käyttöoikeudet
4. Botti on nyt valmis käytettäväksi!

## Komennot

Sosso-Bot tarjoaa seuraavat komennot:

### `/pelipaiva`

Luo viikon pelipäivä-äänestyksen.

**Parametrit:**
- `vihollinen` (pakollinen): Vastustajajoukkueen nimi

**Esimerkki:**
```
/pelipaiva vihollinen:Team XYZ
```

Tämä komento luo äänestyksen, jossa jäsenet voivat äänestää sopivasta pelipäivästä viikolla.

### `/harkka`

Luo harjoituspeli-äänestyksen.

**Parametrit:**
- `kuvaus` (pakollinen): Harjoituspelin kuvaus

**Esimerkki:**
```
/harkka kuvaus:5v5 practice match
```

Käytä tätä komentoa järjestääksesi harjoituspelejä joukkueelle.

### `/unsubscribe`

Lopeta ottelu-ilmoitusten vastaanottaminen kanavalla.

**Parametrit:**
- `liiga` (valinnainen): Liigan nimi muodossa '20 Divisioona S11'

**Esimerkit:**
```
/unsubscribe
/unsubscribe liiga:20 Divisioona S11
```

Jos liigaa ei määritetä, kaikki tilaukset poistetaan kanavalta.

### `/subscriptions`

Näytä ja hallinnoi kanavan ottelu-ilmoitustilauksia.

**Esimerkki:**
```
/subscriptions
```

## Ottelu-ilmoitukset

Botti lähettää automaattisesti ilmoituksia Faceit-otteluista määritetyille kanaville. Ilmoitukset sisältävät:

- Ottelun tiedot (joukkueet, aika)
- Suoran linkin otteluun
- Kartta- ja turnausinfot

## Asennus ja konfigurointi

### Ympäristömuuttujat

Botin käyttöönotto vaatii seuraavat ympäristömuuttujat:

```bash
DISCORD_BOT_TOKEN=your_bot_token_here
DISCORD_MATCH_CHANNEL=your_channel_id_here
```

### Docker-käyttöönotto

Botti voidaan ajaa Docker-kontissa:

```bash
docker build -t sosso-bot .
docker run -e DISCORD_BOT_TOKEN=token -e DISCORD_MATCH_CHANNEL=channel_id sosso-bot
```

### Webhook-konfigurointi

Botti kuuntelee webhook-kutsuja portissa 8080. Varmista, että:

1. Portti 8080 on avoin
2. Faceit-webhook on konfiguroitu osoittamaan botin URL:iin
3. SSL/TLS on käytössä tuotantoympäristössä

## Vianmääritys

### Botti ei vastaa komentoihin

- Varmista, että botilla on oikeat käyttöoikeudet palvelimella
- Tarkista, että botti on online-tilassa
- Yritä poistaa botti palvelimelta ja lisätä se uudelleen

### Ottelu-ilmoituksia ei tule

- Tarkista, että kanava on tilannut ilmoitukset
- Varmista, että webhook on konfiguroitu oikein
- Tarkista botin lokit mahdollisten virheiden varalta

### Äänestykset eivät toimi

- Varmista, että botilla on oikeus lähettää viestejä kanavalle
- Tarkista, että botilla on oikeus lisätä reaktioita viesteihin

## Tekninen dokumentaatio

### Arkkitehtuuri

Sosso-Bot koostuu seuraavista komponenteista:

- **Discord-integraatio**: Käsittelee Discord-komennot ja -vuorovaikutukset
- **Faceit API**: Hakee turnaus- ja ottelutiedot
- **Webhook-palvelin**: Vastaanottaa reaaliaikaisia päivityksiä
- **Tietokanta**: Tallentaa tilaukset ja konfiguraatiot

### Käytetyt teknologiat

- **Go**: Pääohjelmointikieli
- **DiscordGo**: Discord API -kirjasto
- **SQLite**: Tietokantaratkaisu

## Tuki ja palaute

Jos kohtaat ongelmia tai sinulla on ehdotuksia:

1. Tarkista ensin [vianmääritysohje](#vianmääritys)
2. Ota yhteyttä palvelimen järjestelmänvalvojaan
3. Raportoi bugit GitHub-repositoryn Issues-osiossa

## Lisenssi

Tämä projekti on avoimen lähdekoodin projekti. Katso lisätietoja repositorion LICENSE-tiedostosta.

---

**Huomio**: Tämä dokumentaatio on suunnattu loppukäyttäjille. Kehittäjille tarkoitettu dokumentaatio löytyy projektin GitHub-repositoriosta.
