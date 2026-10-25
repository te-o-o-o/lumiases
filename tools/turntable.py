# Render a 3D bust into the frames lumiases plays: a transparent PNG for
# every yaw step and pitch, for the neutral face and for each shape key
# (expression) the mesh has. The head turns under a fixed light, as a real
# head would. A helper to prepare heads, outside the Go build.
#
# blender -b -P tools/turntable.py -- bust.glb heads/name [--step 10] [--pitch -12,0,12] [--size 360]
#
# Frames are named <expression>-y<yaw>-p<pitch>.png, e.g. neutral-y030-p0.png.
import argparse
import math
import os
import sys

import bpy
from mathutils import Matrix, Vector

ap = argparse.ArgumentParser()
ap.add_argument("mesh")
ap.add_argument("out")
ap.add_argument("--step", type=int, default=10, help="degrees between yaw frames")
ap.add_argument("--pitch", default="-12,0,12", help="nod angles in degrees, comma separated")
ap.add_argument("--size", type=int, default=360, help="frame height in pixels")
ap.add_argument("--faces", type=int, default=150_000, help="decimate meshes above this many faces")
ap.add_argument("--rotate", default="", help="turns to stand the model up, applied in order, e.g. z:-90,y:90 (degrees)")
ap.add_argument("--smooth", type=int, default=0, help="smoothing passes, for bumpy generated meshes")
args = ap.parse_args(sys.argv[sys.argv.index("--") + 1 :])

bpy.ops.wm.read_factory_settings(use_empty=True)
ext = os.path.splitext(args.mesh)[1].lower()
{
    ".obj": lambda p: bpy.ops.wm.obj_import(filepath=p),
    ".stl": lambda p: bpy.ops.wm.stl_import(filepath=p),
    ".ply": lambda p: bpy.ops.wm.ply_import(filepath=p),
    ".glb": lambda p: bpy.ops.import_scene.gltf(filepath=p),
    ".gltf": lambda p: bpy.ops.import_scene.gltf(filepath=p),
}[ext](args.mesh)

# One object, centred, 2 units tall, front facing -Y (Blender's front view).
meshes = [o for o in bpy.context.scene.objects if o.type == "MESH"]
bpy.ops.object.select_all(action="DESELECT")
for o in meshes:
    o.select_set(True)
bpy.context.view_layer.objects.active = meshes[0]
if len(meshes) > 1:
    bpy.ops.object.join()
head = bpy.context.view_layer.objects.active
# Apply the importer's own axis conversion first, then --rotate on top.
bpy.ops.object.transform_apply(location=True, rotation=True, scale=True)
for turn in filter(None, args.rotate.split(",")):
    axis, deg = turn.split(":")
    head.rotation_euler = [math.radians(float(deg)) if a == axis else 0 for a in "xyz"]
    bpy.ops.object.transform_apply(rotation=True)
corners = [Vector(c) for c in head.bound_box]
lo = Vector(min(c[i] for c in corners) for i in range(3))
hi = Vector(max(c[i] for c in corners) for i in range(3))
head.data.transform(Matrix.Translation(-(lo + hi) / 2))
s = 2 / (hi.z - lo.z)
head.scale = (s, s, s)
bpy.ops.object.transform_apply(scale=True)
width = max(hi.x - lo.x, hi.y - lo.y) * s

# Scans run to millions of faces; past --faces they only slow the render.
# Blender refuses to apply a modifier to a mesh with shape keys: sculpt
# expressions on an already light mesh.
if len(head.data.polygons) > args.faces and not head.data.shape_keys:
    m = head.modifiers.new("decimate", "DECIMATE")
    m.ratio = args.faces / len(head.data.polygons)
    bpy.ops.object.modifier_apply(modifier=m.name)
if args.smooth:
    m = head.modifiers.new("smooth", "SMOOTH")
    m.factor, m.iterations = 0.6, args.smooth
    bpy.ops.object.modifier_apply(modifier=m.name)
bpy.ops.object.shade_smooth()

# Clay: one plain material, so the light alone draws the forms (a scan's own
# texture bakes in the museum's lighting, which would fight ours).
clay = bpy.data.materials.new("clay")
clay.use_nodes = True
bsdf = clay.node_tree.nodes["Principled BSDF"]
bsdf.inputs["Base Color"].default_value = (0.8, 0.78, 0.74, 1)
bsdf.inputs["Roughness"].default_value = 0.6
head.data.materials.clear()
head.data.materials.append(clay)

# Key from the upper left front, a soft fill, a rim from behind: the rim
# keeps the silhouette readable once it is turned into dots and glyphs.
def light(name, kind, energy, loc):
    d = bpy.data.lights.new(name, kind)
    d.energy = energy
    o = bpy.data.objects.new(name, d)
    o.location = loc
    o.rotation_euler = (Vector((0, 0, 0)) - Vector(loc)).to_track_quat("-Z", "Y").to_euler()
    bpy.context.scene.collection.objects.link(o)

# The key comes from the side so the relief casts shade: a frontal light
# lights every surface alike and the features vanish.
light("key", "SUN", 4.0, (-4, -1.5, 2.5))
light("fill", "SUN", 0.25, (4, -3, 0))
light("rim", "SUN", 2.5, (1, 4, 2))

cam = bpy.data.objects.new("cam", bpy.data.cameras.new("cam"))
cam.data.type = "ORTHO"
cam.data.ortho_scale = max(2.2, width * 1.1)
cam.location = (0, -10, 0)
cam.rotation_euler = (math.pi / 2, 0, 0)
bpy.context.scene.collection.objects.link(cam)

scene = bpy.context.scene
scene.camera = cam
scene.world = bpy.data.worlds.new("world")
scene.world.color = (0.02, 0.02, 0.02)
engines = [e.identifier for e in bpy.types.RenderSettings.bl_rna.properties["engine"].enum_items]
scene.render.engine = next(e for e in ("BLENDER_EEVEE_NEXT", "BLENDER_EEVEE", "CYCLES") if e in engines)
if scene.render.engine == "CYCLES":
    scene.cycles.samples = 32
scene.render.film_transparent = True
scene.render.resolution_x = scene.render.resolution_y = args.size
scene.render.image_settings.file_format = "PNG"
scene.render.image_settings.color_mode = "RGBA"

os.makedirs(args.out, exist_ok=True)
keys = head.data.shape_keys.key_blocks[1:] if head.data.shape_keys else []
pitches = [int(p) for p in args.pitch.split(",")]
for expr in [None] + list(keys):
    for k in keys:
        k.value = 1.0 if k == expr else 0.0
    name = expr.name.lower() if expr else "neutral"
    for yaw in range(0, 360, args.step):
        for pitch in pitches:
            # Yaw turns the head to its left as yaw grows; pitch > 0 nods down.
            head.rotation_euler = (math.radians(pitch), 0, math.radians(yaw))
            scene.render.filepath = os.path.join(args.out, f"{name}-y{yaw:03d}-p{pitch}.png")
            bpy.ops.render.render(write_still=True)
    print(f"{name}: {360 // args.step * len(pitches)} frames")
