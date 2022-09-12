using UnityEngine.SceneManagement;

public enum SceneType {
    Title,
    Game,
}
public class SceneController
{

    public static void LoadScene(SceneType scene)
    {
        SceneManager.LoadScene(scene.ToString() + "Scene");
    }
}
