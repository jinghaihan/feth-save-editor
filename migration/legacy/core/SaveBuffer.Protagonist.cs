using System;
using System.Linq;

namespace FethEditor.Core
{
    public sealed partial class SaveBuffer
    {
        private int? ProtagonistSlot()
        {
            var slots = Data.Characters.Select((character, index) => (character, index))
                .Where(row => (row.character.data.Id is 0 or 1) && row.character.data.Level > 0)
                .Select(row => row.index).ToArray();
            if (slots.Length > 1)
                throw new InvalidOperationException("The save contains ambiguous protagonist records.");
            return slots.Length == 1 ? slots[0] : null;
        }

        public int? ReadProtagonistGender()
        {
            int? slot = ProtagonistSlot();
            return slot.HasValue ? Data.Characters[slot.Value].data.Id : null;
        }

        public void SetProtagonistGender(int gender)
        {
            if (gender is not (0 or 1))
                throw new ArgumentOutOfRangeException(nameof(gender), "Gender must be 0 (male) or 1 (female).");
            int slot = ProtagonistSlot() ?? throw new InvalidOperationException("The save has no protagonist record.");
            Set($"Characters[{slot}].data.Id", gender);
        }
    }
}
