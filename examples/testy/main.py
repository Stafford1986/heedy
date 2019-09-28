import asyncio
from aiohttp import web

from heedy import Plugin

p = Plugin()

routes = web.RouteTableDef()


@routes.post("/cupdate")
async def index(request):
    print("REQUEST", request.headers)
    print("GOT", await request.json())
    return web.Response(text="OK")


@routes.post("/supdate")
async def index(request):
    print("REQUEST", request.headers)
    print("GOT", await request.json())

    return web.Response(text="OK")


@routes.get("/api/testy/lol")
async def lol(request):
    myc = await p.listConnections(avatar="false", plugin="testy:tree")
    await p.notify({"key": "lol", "type": "info", "title": "Test Notification", "description": "A description", "connection": myc[0]["id"]})
    return web.Response(text="lol")


@routes.get("/api/testy/lol2")
async def lol(request):
    myc = await p.listConnections(avatar="false", plugin="testy:tree")
    await p.delete_notification({"key": "lol", "connection": myc[0]["id"]})
    return web.Response(text="lol")

app = web.Application()
app.add_routes(routes)


async def runme():
    await p.fire({
        "user": "test",
        "event": "LOL"
    })

    # Runs the server over a unix domain socket. The socket is automatically placed in the data folder,
    # and not the plugin folder.
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.UnixSite(runner, path=f"{p.name}.sock")
    await site.start()
    print("Plugin Ready")


asyncio.ensure_future(runme())
asyncio.get_event_loop().run_forever()
