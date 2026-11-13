// Development-only conversion of the existing editor's game data and packed layouts.
using System.IO.Compression;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Text.Json;
using FethEditor.Core;
using SaveEditor;
using SaveEditor.Structs;

var output = Path.GetFullPath(args[0]);
Directory.CreateDirectory(output);
var json = new JsonSerializerOptions { IncludeFields = true };
void Write(string name, object value) {
    using var file = File.Create(Path.Combine(output, name + ".json.gz"));
    using var gzip = new GZipStream(file, CompressionLevel.SmallestSize);
    JsonSerializer.Serialize(gzip, value, json);
}
Write("ui-zh_hans", JsonSerializer.Deserialize<Dictionary<string,string>>(File.ReadAllText("migration/legacy/ui-localization/zh_hans.json"))!);
var schemas = new Dictionary<string, object>();
void Schema(Type type) {
    if (schemas.ContainsKey(type.Name)) return;
    var fields = type.GetFields(BindingFlags.Public | BindingFlags.Instance);
    schemas[type.Name] = new { size = Marshal.SizeOf(type), fields = fields.Select(f => {
        var t = f.FieldType.IsArray ? f.FieldType.GetElementType()! : f.FieldType;
        return new { name = f.Name, offset = (int)Marshal.OffsetOf(type, f.Name), type = t.Name,
            length = f.FieldType.IsArray ? f.GetCustomAttribute<MarshalAsAttribute>()!.SizeConst : 0,
            size = Marshal.SizeOf(t) };
    }).ToArray() };
    foreach (var f in fields) {
        var t = f.FieldType.IsArray ? f.FieldType.GetElementType()! : f.FieldType;
        if (!t.IsPrimitive && !t.IsEnum) Schema(t);
    }
}
Schema(typeof(SaveData_V23)); Schema(typeof(SystemSaveData_V7));
Write("schema", schemas);
string Safe(Func<string> f) { try { return f(); } catch { return "???"; } }
foreach (var language in Enum.GetValues<enmLanguage>()) {
    Database.Init(language);
    var db = Database.BinaryDatabase;
    var catalog = new Dictionary<string, object>();
    foreach (var pair in new Dictionary<string, Dictionary<int,string>> {
        ["items"] = Database.ItemList, ["characters"] = Database.UnitList,
        ["classes"] = Database.ClassList, ["battalions"] = Database.BattalionList,
        ["battalion-skills"] = Database.BattalionSkillList, ["abilities"] = Database.AbilityList,
        ["arts"] = Database.CombatArtList, ["magic"] = Database.MagicList })
        catalog[pair.Key] = pair.Value.Select(e => new { id = e.Key, name = e.Value,
            mainCharacterId = pair.Key == "characters" && e.Key >= 0 ? db.CharacterEntries[e.Key].MainCharacterId : -1 }).ToArray();
    catalog["misc"] = Enumerable.Range(0,Player_V23.COUNT_MISC_ITEMS).Select(id => new { id, name = Safe(() => Database.GetMiscItemName(id)) }).ToArray();
    catalog["gifts"] = Enumerable.Range(0,Player_V23.COUNT_GIFT_ITEMS).Select(id => new { id, name = Safe(() => Database.GetGiftItemName(id)) }).ToArray();
    catalog["quests"] = Enumerable.Range(0,150).Select(id => new { id, name = Safe(() => Database.GetQuestName(id)) }).ToArray();
    catalog["supports"] = Enumerable.Range(0,270).Select(id => new { id, name = Safe(() => Database.GetSupportTalkName(id)), ranks = SupportPairRanks.AvailableRanks(id), maximumRank = SupportPairRanks.MaxRank(id) }).ToArray();
    catalog["support-ranks"] = SupportRankPresets.Values.Select(v => new { rank=v.Name, points=v.Points }).ToArray();
    catalog["obtainable-battalions"] = ObtainableBattalions.All.Select(v => new { id=v.Type, name=Database.GetBattalionName(v.Type), experience=400, stamina=v.Stamina, skill=v.Skill }).ToArray();
    var details = new Dictionary<string, object> {
        ["characters"] = Enumerable.Range(0,db.CharacterEntries.Count).Select(id => new { id, name=Safe(() => Database.GetUnitName(id)), details=db.CharacterEntries[id].GenerateDebugOut() }).ToArray(),
        ["classes"] = Enumerable.Range(0,db.ClassEntries.Count).Select(id => new { id, name=Safe(() => Database.GetClassName(id)), details=db.ClassEntries[id].GenerateDebugOut() }).ToArray(),
        ["items"] = db.ItemEntries.Select(e => new { id=e.Key, name=Database.GetItemName(e.Key), details=e.Value.GenerateDebugOut() }).ToArray()
    };
    var strings = new Dictionary<string,string>();
    foreach(var i in new[]{652,1153,651,970,3937,1163,461,893,569,260,264,2324,2326,1814,1807,385,1811,1060}) strings[i.ToString()] = Safe(() => Database.GetString(i,1));
    catalog["system-flags"] = Enumerable.Range(0,SystemSaveData_V7.COUNT_FLAGS).Select(id => new { id, name=id>=8 && id<108 ? Safe(() => Database.GetString(12823+id-8)) : "" }).ToArray();
    var journal = new NgPlusJournal(new byte[0x2000],0);
    catalog["records"] = Enumerable.Range(0,45).Select(id=>new {id,name=journal.GetCharacterName(id)}).ToArray();
    var names = new Dictionary<string, Dictionary<int,string>> {
        ["characters"] = Enumerable.Range(0,db.CharacterEntries.Count).ToDictionary(id=>id,id=>Safe(()=>Database.GetUnitName(id))),
        ["items"] = db.ItemEntries.Keys.ToDictionary(id=>id,id=>Safe(()=>Database.GetItemName(id))),
        ["abilities"] = Enumerable.Range(0,256).ToDictionary(id=>id,id=>Safe(()=>Database.GetAbilityName(id,false))),
        ["unit-classes"] = Enumerable.Range(0,64).ToDictionary(id=>id,id=>Safe(()=>Database.GetUnitClassName(id)))
    };
    foreach(var type in new[]{typeof(enmCharacterFlags),typeof(enmCharacterClassFlags)})
        names[type.Name] = Enumerable.Range(0,type==typeof(enmCharacterFlags)?32:8).ToDictionary(id=>id,id=>{
            var entry=Enum.GetName(type,Enum.ToObject(type,1u<<id));
            return entry==null ? $"Bit {id}" : type.GetField(entry)?.GetCustomAttribute<System.ComponentModel.DescriptionAttribute>()?.Description ?? entry;
        });
    names["ui"] = new[]{1025,1026,1027,1212,405,402,401,555,1860,3696,1820}
        .Concat(Enumerable.Range(373,Database.DIFFICULTY_COUNT)).Concat(Enumerable.Range(381,Database.GAMESTYLE_COUNT))
        .ToDictionary(id=>id,id=>Safe(()=>Database.GetString(id,1)));
    foreach(var id in Enumerable.Range(9764,4).Concat(Enumerable.Range(7214,11)))
        names["ui"][id]=Safe(()=>Database.GetString(id));
    Write(language.ToString(), new { catalog, details, strings, names });
}
Database.Init(enmLanguage.en_u);
var binary = Database.BinaryDatabase;
Write("rules",new {
    constants=typeof(Database).GetFields(BindingFlags.Public|BindingFlags.Static).Where(f=>f.IsLiteral && f.FieldType==typeof(int)).ToDictionary(f=>f.Name,f=>f.GetValue(null)),
    skillThresholds=Database.SkillLevelupRank, professorThresholds=Database.TeacherLevelupRank,
    essentialItems=Database.EssentialItems.Distinct(),
    units=binary.CharacterEntries.Select((v,id)=>new { id, record=ClassEligibility.RecordForUnit(id), maximumHP=v.MaximumHP, maximumStats=v.MaximumStats }),
    classes=binary.ClassEntries.Select((v,id)=>new { id, baseHP=v.BaseHP, baseStats=v.BaseStats, maxExp=Database.GetMaxClassExp(id) }),
    durability=binary.ItemEntries.ToDictionary(e=>e.Key,e=>(int)e.Value.Durability),
    eligible=Enumerable.Range(0,45).Select(r=>Enumerable.Range(0,100).Where(c=>ClassEligibility.IsAvailable(r,c)).ToArray()),
    playable=Enumerable.Range(0,45).Where(ClassEligibility.IsPlayableRecord).ToArray()
});
Console.WriteLine("Exported packed layouts and all game languages.");
