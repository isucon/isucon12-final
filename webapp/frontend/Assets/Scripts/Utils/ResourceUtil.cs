using Data;
using UnityEngine;

public static class ResourceUtil
{
    public static Sprite LoadIcon(this ItemMaster item)
    {
        switch (item.item_type)
        {
            case (int)ItemType.Coin:
            {;
                return Resources.Load<Sprite>("Textures/icon_coin");
            }
            case (int)ItemType.Hammer:
            {
                const string basePath = "Textures/Items/item_hummer_";
                var hummerId = item.id - 1;
                var path = $"{basePath}{hummerId:D2}";
                return Resources.Load<Sprite>(path);
            }
            case (int)ItemType.Exp:
            {
                const string basePath = "Textures/Items/item_portion_";
                var portionId = item.id - 16;
                var path = $"{basePath}{portionId:D2}";
                return Resources.Load<Sprite>(path);
            }
            case (int)ItemType.Timer:
            {
                const string basePath = "Textures/Items/item_timer_";
                var timerId = item.id - 20;
                var path = $"{basePath}{timerId:D2}";
                return Resources.Load<Sprite>(path);
            }
        }

        return null;
    }
}
