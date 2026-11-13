using System;
using System.Collections.Generic;
using System.Linq;
using SaveEditor;
using SaveEditor.Structs;

namespace FethEditor.Core
{
    public sealed partial class SaveBuffer
    {
        // v1.2.0 (89048449BA238C8CF565518B83BF02D3): level = Exp / 100 + 1
        // at main+0x5D177C; battle experience is capped at 400 at main+0x0D5200.
        public const ushort MaximumBattalionExperience = 400;

        public bool MaximizeBattalionLevel(int slot)
        {
            SaveData_V23 data = Data;
            int? equippedSlot = EquippedBattalionSlot(data, slot);
            if (data.Player.Battalions[slot].Type >= Database.BATTALION_COUNT)
                throw new InvalidOperationException("No battalion is selected.");
            var paths = new List<string> { $"Player.Battalions[{slot}].Exp" };
            if (equippedSlot.HasValue)
                paths.Add($"Characters[{equippedSlot.Value}].data.EquippedBattalion.Exp");
            return MaximizeBattalionExperience(paths) > 0;
        }

        public int MaximizeBattalionLevels()
        {
            SaveData_V23 data = Data;
            var paths = new List<string>();
            for (int slot = 0; slot < data.Player.Battalions.Length; slot++)
            {
                if (data.Player.Battalions[slot].Type >= Database.BATTALION_COUNT) continue;
                // Validate every equipment link before applying any part of the batch.
                EquippedBattalionSlot(data, slot);
                paths.Add($"Player.Battalions[{slot}].Exp");
            }
            for (int slot = 0; slot < data.Characters.Length; slot++)
            {
                CharacterData_V23 character = data.Characters[slot].data;
                if (character.Id >= 0 && character.Level > 0
                    && character.EquippedBattalion.Type < Database.BATTALION_COUNT)
                    paths.Add($"Characters[{slot}].data.EquippedBattalion.Exp");
            }
            return MaximizeBattalionExperience(paths);
        }

        private int MaximizeBattalionExperience(IEnumerable<string> paths)
        {
            var locations = paths.Where(path => (long)Get(path) < MaximumBattalionExperience)
                .Select(Resolve).ToArray();
            foreach (var location in locations) WriteNumber(location, MaximumBattalionExperience);
            return locations.Length;
        }

        public void DeleteBattalion(int slot)
        {
            SaveData_V23 data = Data;
            int? equippedSlot = EquippedBattalionSlot(data, slot);
            if (data.Player.Battalions[slot].Type >= Database.BATTALION_COUNT)
                throw new InvalidOperationException("No battalion is selected.");
            var prefixes = new List<string> { $"Player.Battalions[{slot}]." };
            if (equippedSlot.HasValue)
                prefixes.Add($"Characters[{equippedSlot.Value}].data.EquippedBattalion.");
            var fields = new (string Name, long Value)[]
            {
                ("CharacterId", -1), ("Exp", 0), ("Stamina", 0),
                ("Type", Database.BATTALION_COUNT), ("Skill", Database.BATTALION_SKILL_COUNT)
            };
            var writes = prefixes.SelectMany(prefix => fields.Select(field =>
                (Location: Resolve(prefix + field.Name), field.Value))).ToArray();
            foreach (var write in writes) WriteNumber(write.Location, write.Value);
        }

        public ushort? GetEquippedBattalionEndurance(int slot)
        {
            SaveData_V23 data = Data;
            int? equippedSlot = EquippedBattalionSlot(data, slot);
            return equippedSlot.HasValue
                ? data.Characters[equippedSlot.Value].data.EquippedBattalion.Stamina : null;
        }

        public void SetBattalionEnduranceValues(int slot, ushort storedEndurance, ushort? equippedEndurance = null)
        {
            SaveData_V23 data = Data;
            if (slot < 0 || slot >= data.Player.Battalions.Length)
                throw new ArgumentOutOfRangeException(nameof(slot));
            var writes = new List<(string Path, ushort Endurance)>
            {
                ($"Player.Battalions[{slot}].Stamina", storedEndurance)
            };
            if (equippedEndurance.HasValue)
            {
                int equippedSlot = EquippedBattalionSlot(data, slot)
                    ?? throw new InvalidOperationException("This battalion is not equipped by an active character.");
                writes.Add(($"Characters[{equippedSlot}].data.EquippedBattalion.Stamina", equippedEndurance.Value));
            }
            var locations = writes.Select(write => (Location: Resolve(write.Path), write.Endurance)).ToArray();
            foreach (var write in locations) WriteNumber(write.Location, write.Endurance);
        }

        public ushort GetBattalionEndurance(int slot)
        {
            SaveData_V23 data = Data;
            int? equippedSlot = EquippedBattalionSlot(data, slot);
            return equippedSlot.HasValue
                ? data.Characters[equippedSlot.Value].data.EquippedBattalion.Stamina
                : data.Player.Battalions[slot].Stamina;
        }

        public void SetBattalionEndurance(int slot, ushort endurance)
        {
            SaveData_V23 data = Data;
            int? equippedSlot = EquippedBattalionSlot(data, slot);
            string[] paths = equippedSlot.HasValue
                ? [$"Player.Battalions[{slot}].Stamina", $"Characters[{equippedSlot.Value}].data.EquippedBattalion.Stamina"]
                : [$"Player.Battalions[{slot}].Stamina"];
            var locations = paths.Select(Resolve).ToArray();
            foreach (var location in locations) WriteNumber(location, endurance);
        }

        public bool ReplenishCharacterBattalion(int slot)
        {
            SaveData_V23 data = Data;
            if (slot < 0 || slot >= data.Characters.Length)
                throw new ArgumentOutOfRangeException(nameof(slot));
            CharacterData_V23 character = data.Characters[slot].data;
            if (character.Id < 0 || character.Level == 0)
                throw new InvalidOperationException("No active character is selected.");
            ushort full = ObtainableBattalions.FullEndurance(character.EquippedBattalion.Type)
                ?? throw new InvalidOperationException("Full endurance is unknown for this battalion type.");
            int[] inventorySlots = Enumerable.Range(0, data.Player.Battalions.Length)
                .Where(index => data.Player.Battalions[index].CharacterId == character.Id
                    && data.Player.Battalions[index].Type == character.EquippedBattalion.Type).ToArray();
            if (inventorySlots.Length > 0)
            {
                if (inventorySlots.Length != 1 || EquippedBattalionSlot(data, inventorySlots[0]) != slot)
                    throw new InvalidOperationException("The equipped battalion cannot be uniquely matched to its barracks entry.");
                return ReplenishBattalion(inventorySlots[0]);
            }
            bool changed = character.EquippedBattalion.Stamina != full;
            WriteNumber(Resolve($"Characters[{slot}].data.EquippedBattalion.Stamina"), full);
            return changed;
        }

        public bool ReplenishBattalion(int slot)
        {
            SaveData_V23 data = Data;
            int? equippedSlot = EquippedBattalionSlot(data, slot);
            ushort full = ObtainableBattalions.FullEndurance(data.Player.Battalions[slot].Type)
                ?? throw new InvalidOperationException("Full endurance is unknown for this battalion type.");
            bool changed = data.Player.Battalions[slot].Stamina != full
                || equippedSlot.HasValue
                    && data.Characters[equippedSlot.Value].data.EquippedBattalion.Stamina != full;
            SetBattalionEndurance(slot, full);
            return changed;
        }

        public (int Replenished, int Skipped) ReplenishBattalions()
        {
            SaveData_V23 data = Data;
            var writes = new List<(string Path, ushort Endurance)>();
            var linkedCharacters = new HashSet<int>();
            int replenished = 0, skipped = 0;
            for (int slot = 0; slot < data.Player.Battalions.Length; slot++)
            {
                Battalion battalion = data.Player.Battalions[slot];
                if (battalion.Type == Database.BATTALION_COUNT) continue;
                int? equippedSlot = EquippedBattalionSlot(data, slot);
                if (equippedSlot.HasValue) linkedCharacters.Add(equippedSlot.Value);
                ushort? full = ObtainableBattalions.FullEndurance(battalion.Type);
                if (!full.HasValue)
                {
                    skipped++;
                    continue;
                }
                bool changed = battalion.Stamina != full.Value;
                writes.Add(($"Player.Battalions[{slot}].Stamina", full.Value));
                if (equippedSlot.HasValue)
                {
                    changed |= data.Characters[equippedSlot.Value].data.EquippedBattalion.Stamina != full.Value;
                    writes.Add(($"Characters[{equippedSlot.Value}].data.EquippedBattalion.Stamina", full.Value));
                }
                if (changed) replenished++;
            }

            // Also cover equipped battalions without a matching barracks entry.
            for (int slot = 0; slot < data.Characters.Length; slot++)
            {
                CharacterData_V23 character = data.Characters[slot].data;
                if (linkedCharacters.Contains(slot) || character.Id < 0 || character.Level == 0
                    || character.EquippedBattalion.Type == Database.BATTALION_COUNT) continue;
                ushort? full = ObtainableBattalions.FullEndurance(character.EquippedBattalion.Type);
                if (!full.HasValue)
                {
                    skipped++;
                    continue;
                }
                if (character.EquippedBattalion.Stamina != full.Value) replenished++;
                writes.Add(($"Characters[{slot}].data.EquippedBattalion.Stamina", full.Value));
            }

            // Resolve every write first so ambiguous links cannot partially apply a batch.
            var locations = writes.Select(write => (Location: Resolve(write.Path), write.Endurance)).ToArray();
            foreach (var write in locations) WriteNumber(write.Location, write.Endurance);
            return (replenished, skipped);
        }

        private static int? EquippedBattalionSlot(SaveData_V23 data, int slot)
        {
            if (slot < 0 || slot >= data.Player.Battalions.Length)
                throw new ArgumentOutOfRangeException(nameof(slot));
            Battalion battalion = data.Player.Battalions[slot];
            if (battalion.CharacterId < 0 || battalion.Type == Database.BATTALION_COUNT) return null;
            int[] characters = data.Characters.Select((character, index) => (character.data, index))
                .Where(entry => entry.data.Id == battalion.CharacterId && entry.data.Level > 0
                    && entry.data.EquippedBattalion.Type == battalion.Type)
                .Select(entry => entry.index).ToArray();
            if (characters.Length == 0) return null;
            if (characters.Length != 1 || data.Player.Battalions.Count(other =>
                other.CharacterId == battalion.CharacterId && other.Type == battalion.Type) != 1)
                throw new InvalidOperationException("The equipped battalion cannot be uniquely matched to its barracks entry.");
            return characters[0];
        }
    }
}
